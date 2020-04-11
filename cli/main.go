package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/meisterluk/dupfiles-go/internals"
	v1 "github.com/meisterluk/dupfiles-go/v1"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var app *kingpin.Application
var report *cliReportCommand
var find *cliFindCommand
var stats *cliStatsCommand
var digest *cliDigestCommand
var diff *cliDiffCommand
var hashAlgos *cliHashAlgosCommand
var version *cliVersionCommand

const usageTemplate = `{{define "FormatCommand"}}\
{{if .FlagSummary}} {{.FlagSummary}}{{end}}\
{{range .Args}} {{if not .Required}}[{{end}}<{{.Name}}>{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}\
{{end}}\


{{define "FormatCommands"}}\
{{range .FlattenedCommands}}\
{{if not .Hidden}}\
  ["{{.FullCommand}}", "{{if .Default}}*{{end}}{{template "FormatCommand" .}}",
{{.Help|Wrap 4}}
{{end}}\
{{end}}\
{{end}}\

{{define "FormatUsage"}}\
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}\
{{end}}\

{
{{if .Context.SelectedCommand}}\
  "usage": "{{.App.Name}} {{.Context.SelectedCommand}}{{template "FormatUsage" .Context.SelectedCommand}}",
{{if .Context.SelectedCommand.Help}}\
  "help": "{{.Context.SelectedCommand.Help}}",
{{end}}\
{{else}}\
  "usage": "{{.App.Name}}{{template "FormatUsage" .App}}",
  "help": "{{.App.Help}}",
{{end}}\
{{if .Context.Flags}}\
  "flags": [
{{range .Context.Flags}}{{if not .Hidden}}\
    ["{{.|FormatFlag true}}", "{{.Help}}"],
{{end}}{{end}}\
  ],
{{end}}\

{{if .Context.Args}}\
  "args": [
{{range .Context.Args}}\
    ["{{if not .Required}}[{{end}}<{{ .Name }}>{{if not .Required}}]{{end}}", "{{.Help}}"],
{{end}}\
  ]
{{end}}\

{{if .Context.SelectedCommand}}\
{{if len .Context.SelectedCommand.Commands}}\
  "subcommands": [
  {{template "FormatCommands" .Context.SelectedCommand}}
]
{{end}}\
{{else if .App.Commands}}\
  "commands": [
  {{template "FormatCommands" .App}}
]
  {{end}}\
}
`

const E_EXISTS = `file '%s' already exists and --overwrite was not specified`
const E_CONFIG_JSON = `could not serialize config JSON: %s`
const E_RESULT_JSON = `could not serialize result JSON: %s`

// CLI response for errors
type errorResponse struct {
	ErrorMessage string `json:"error"`
	ExitCode     int    `json:"-"`
}

func (e *errorResponse) Print() int {
	if jsonOutput() {
		fmt.Fprintf(os.Stderr, "%s\n", e.JSON())
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", e.String())
	}
	return e.ExitCode
}

func (e *errorResponse) String() string {
	return `cli: error: ` + e.ErrorMessage
}

func (e *errorResponse) JSON() string {
	jsonBytes, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON marshalling error: %s", err)
		return ""
	}
	return string(jsonBytes)
}

func init() {
	app = kingpin.New("dupfiles", "Determine duplicate files and folders.")
	app.Version("1.0.0").Author("meisterluk")
	app.HelpFlag.Short('h')

	// if --json, show help as JSON
	if jsonOutput() {
		app.UsageTemplate(usageTemplate)
	} else {
		app.UsageTemplate(kingpin.CompactUsageTemplate)
	}

	report = newCLIReportCommand(app)
	find = newCLIFindCommand(app)
	stats = newCLIStatsCommand(app)
	digest = newCLIDigestCommand(app)
	diff = newCLIDiffCommand(app)
	hashAlgos = newCLIHashAlgosCommand(app)
	version = newCLIVersionCommand(app)
}

func cli(w Output, log Output) (int, bool, error) {
	// <profiling> TODO
	/*f, err := os.Create("cpu.prof")
	if err != nil {
		w.Println(err)
		return 200
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()*/
	// </profiling>

	subcommand, err := app.Parse(os.Args[1:])

	if err != nil {
		return 1, internals.Contains(os.Args[1:], "--json"), err
	}

	// TODO verify that all input files are properly UTF-8 encoded ⇒ output if properly UTF-8 encoded

	var jsonOutput bool
	switch subcommand {
	case report.cmd.FullCommand():
		reportSettings, err := report.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}
		jsonOutput = reportSettings.JSONOutput

		// config output
		if reportSettings.ConfigOutput {
			b, err := json.Marshal(reportSettings)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_CONFIG_JSON, err)
			}

			w.Println(string(b))
			return 0, reportSettings.JSONOutput, nil
		}

		// TODO: implement continue option

		// consider reportSettings.Overwrite
		_, err = os.Stat(reportSettings.Output)
		if err == nil && !reportSettings.Overwrite {
			return 3, jsonOutput, fmt.Errorf(E_EXISTS, reportSettings.Output)
		}

		// create report
		rep, err := internals.NewReportWriter(reportSettings.Output)
		if err != nil {
			return 2, jsonOutput, fmt.Errorf(`error writing file '%s': %s`, reportSettings.Output, err)
		}
		// NOTE since we create a file descriptor for the output file here already,
		//      we need to exclude it from the walk finding all paths.
		//      We could move file descriptor creation to a later point, but I want
		//      to catch FS writing issues early.
		reportSettings.ExcludeTree = append(reportSettings.ExcludeTree, reportSettings.Output)

		fullPath, err := filepath.Abs(reportSettings.BaseNode)
		if err != nil {
			return 6, jsonOutput, err
		}
		err = rep.HeadLine(reportSettings.HashAlgorithm, !reportSettings.EmptyMode, reportSettings.BaseNodeName, fullPath)
		if err != nil {
			return 6, jsonOutput, err
		}

		// walk and write tail lines
		entries := make(chan internals.ReportTailLine)
		errChan := make(chan error)
		go internals.HashATree(
			reportSettings.BaseNode, reportSettings.DFS, reportSettings.IgnorePermErrors, reportSettings.HashAlgorithm,
			reportSettings.ExcludeBasename, reportSettings.ExcludeBasenameRegex, reportSettings.ExcludeTree,
			reportSettings.BasenameMode, reportSettings.Workers, entries, errChan,
		)

		for entry := range entries {
			err = rep.TailLine(entry.HashValue, entry.NodeType, entry.FileSize, entry.Path)
			if err != nil {
				return 2, jsonOutput, err
			}
		}

		err, ok := <-errChan
		if ok {
			// TODO proper exit code required
			return 6, jsonOutput, err
		}

		msg := fmt.Sprintf(`Done. File "%s" written`, reportSettings.Output)
		if reportSettings.JSONOutput {
			type output struct {
				Message string `json:"message"`
			}

			data := output{Message: msg}
			jsonRepr, err := json.Marshal(&data)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
			}

			w.Println(string(jsonRepr))
		} else {
			w.Println(msg)
		}

		return 0, jsonOutput, nil

	case find.cmd.FullCommand():
		findSettings, err := find.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}
		jsonOutput = findSettings.JSONOutput

		if findSettings.ConfigOutput {
			// config output is printed in JSON independent of findSettings.JSONOutput
			b, err := json.Marshal(findSettings)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_CONFIG_JSON, err)
			}

			w.Println(string(b))
			return 0, jsonOutput, nil
		}

		// consider findSettings.Overwrite
		_, err = os.Stat(findSettings.Output)
		if err == nil && !findSettings.Overwrite {
			return 3, jsonOutput, fmt.Errorf(E_EXISTS, findSettings.Output)
		}

		errChan := make(chan error)
		dupEntries := make(chan internals.DuplicateSet)
		exitCode := 0 // TODO requires better feedback from errChan

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			// error goroutine
			defer wg.Done()
			for err := range errChan {
				log.Printfln(`error: %s`, err)
			}
			// TODO is this proper error handling? is the exit code properly propagated?
			// TODO JSON output support
		}()
		go func() {
			// duplicates goroutine
			defer wg.Done()
			type jsonOut struct {
				LineNo     uint64 `json:"lineno"`
				ReportFile string `json:"report"`
				Path       string `json:"path"`
			}

			if findSettings.JSONOutput {
				for entry := range dupEntries {
					// prepare data structure
					entries := make([]jsonOut, 0, len(entry.Set))
					for _, equiv := range entry.Set {
						entries = append(entries, jsonOut{
							LineNo:     equiv.Lineno,
							ReportFile: equiv.ReportFile,
							Path:       equiv.Path,
						})
					}

					// marshal to JSON
					jsonDump, err := json.Marshal(&entries)
					if err != nil {
						log.Printfln(`error marshalling result: %s`, err.Error())
						// TODO? return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
						continue
					}

					w.Println(string(jsonDump))
				}

			} else {
				for entry := range dupEntries {
					//log.Println("<duplicates>")
					out := hex.EncodeToString(entry.Digest) + "\n"
					for _, s := range entry.Set {
						out += `  ` + s.ReportFile + " " + string(filepath.Separator) + " " + s.Path + "\n"
					}
					w.Println(out) // TODO or findSettings.Output
					// TODO json output support
					//log.Println("</duplicates>")
				}
			}
		}()

		internals.FindDuplicates(findSettings.Reports, dupEntries, errChan)
		wg.Wait()

		// TODO: print debug.GCStats ?
		return exitCode, jsonOutput, nil

	case stats.cmd.FullCommand():
		statsSettings, err := stats.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}
		jsonOutput = statsSettings.JSONOutput

		if statsSettings.ConfigOutput {
			// config output is printed in JSON independent of statsSettings.JSONOutput
			b, err := json.Marshal(statsSettings)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_CONFIG_JSON, err)
			}
			w.Println(string(b))
			return 0, jsonOutput, nil
		}

		type sizeEntry struct {
			Path string `json:"path"`
			Size uint64 `json:"size"`
		}

		// BriefReportStatistics contains statistics collected from
		// a report file and only requires single-pass parsing and
		// constant memory to evaluate those statistics
		type BriefReportStatistics struct {
			HeadVersion         [3]uint16     `json:"head-version"`
			HeadTimestamp       time.Time     `json:"head-timestamp"`
			HeadHashAlgorithm   string        `json:"head-hash-algorithm"`
			HeadBasenameMode    bool          `json:"head-basename-mode"`
			HeadNodeName        string        `json:"head-node-name"`
			HeadBasePath        string        `json:"head-base-path"`
			NumUNIXDeviceFile   uint32        `json:"count-unix-device"`
			NumDirectory        uint32        `json:"count-directory"`
			NumRegularFile      uint32        `json:"count-regular-file"`
			NumLink             uint32        `json:"count-link"`
			NumFIFOPipe         uint32        `json:"count-fifo-pipe"`
			NumUNIXDomainSocket uint32        `json:"count-unix-socket"`
			MaxDepth            uint16        `json:"fs-depth-max"`
			TotalSize           uint64        `json:"fs-size-total"`
			Top10MaxSizeFiles   [10]sizeEntry `json:"files-size-max-top10"`
		}
		// BriefReportStatistics contains statistics collected from
		// a report file and requires linear time and linear memory
		// to evaluate those statistics
		type LongReportStatistics struct {
			// {average, median, min, max} number of children in a folder?
		}

		rep, err := internals.NewReportReader(statsSettings.Report)
		if err != nil {
			return 1, jsonOutput, fmt.Errorf(`failure reading report file '%s': %s`, statsSettings.Report, err)
		}
		var briefStats BriefReportStatistics
		for {
			tail, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				rep.Close()
				return 9, jsonOutput, fmt.Errorf(`failure reading report file '%s' tailline: %s`, statsSettings.Report, err)
			}

			// consider node type
			switch tail.NodeType {
			case 'D':
				briefStats.NumDirectory++
			case 'C':
				briefStats.NumUNIXDeviceFile++
			case 'F':
				briefStats.NumRegularFile++
			case 'L':
				briefStats.NumLink++
			case 'P':
				briefStats.NumFIFOPipe++
			case 'S':
				briefStats.NumUNIXDomainSocket++
			default:
				return 9, jsonOutput, fmt.Errorf(`unknown node type '%c'`, tail.NodeType)
			}

			// consider folder depth
			depth := internals.DetermineDepth(tail.Path)
			if depth > briefStats.MaxDepth {
				briefStats.MaxDepth = depth
			}

			// consider size
			briefStats.TotalSize += tail.FileSize
			oldTotalSize := briefStats.TotalSize
			if oldTotalSize > briefStats.TotalSize {
				return 11, jsonOutput, fmt.Errorf(`total-size overflowed from %d to %d`, oldTotalSize, briefStats.TotalSize)
			}

			for i := 0; i < 10; i++ {
				if tail.NodeType == 'D' {
					continue
				}
				if briefStats.Top10MaxSizeFiles[i].Size > tail.FileSize {
					continue
				}
				tmp := briefStats.Top10MaxSizeFiles[i]
				briefStats.Top10MaxSizeFiles[i].Size = tail.FileSize
				briefStats.Top10MaxSizeFiles[i].Path = tail.Path
				for j := i + 1; j < 10; j++ {
					tmp2 := briefStats.Top10MaxSizeFiles[j]
					briefStats.Top10MaxSizeFiles[j] = tmp
					tmp = tmp2
				}
				break
			}
		}

		// report Head data
		briefStats.HeadVersion = rep.Head.Version
		briefStats.HeadTimestamp = rep.Head.Timestamp
		briefStats.HeadHashAlgorithm = rep.Head.HashAlgorithm
		briefStats.HeadBasenameMode = rep.Head.BasenameMode
		briefStats.HeadNodeName = rep.Head.NodeName
		briefStats.HeadBasePath = rep.Head.BasePath

		var longStats LongReportStatistics
		if statsSettings.Long {
			// which data will be evaluated here?
		}

		type resultJSON struct {
			Brief BriefReportStatistics `json:"brief"`
			Long  LongReportStatistics  `json:"long"`
		}
		var out resultJSON
		out.Brief = briefStats
		out.Long = longStats

		if jsonOutput {
			jsonRepr, err := json.Marshal(&out)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
			}
			w.Println(string(jsonRepr))
		} else {
			jsonRepr, err := json.MarshalIndent(&out, "", "  ")
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
			}
			w.Println(string(jsonRepr))
		}

		rep.Close()

	case digest.cmd.FullCommand():
		hashSettings, err := digest.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}
		jsonOutput = hashSettings.JSONOutput

		if hashSettings.ConfigOutput {
			// config output is printed in JSON independent of hashSettings.JSONOutput
			b, err := json.Marshal(hashSettings)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_CONFIG_JSON, err)
			}
			w.Println(string(b))
			return 0, jsonOutput, nil
		}

		fileinfo, err := os.Stat(hashSettings.BaseNode)
		if err != nil {
			return 6, jsonOutput, err
		}

		if fileinfo.IsDir() {
			// generate fsstats concurrently
			stats := internals.GenerateStatistics(hashSettings.BaseNode, hashSettings.IgnorePermErrors, hashSettings.ExcludeBasename, hashSettings.ExcludeBasenameRegex, hashSettings.ExcludeTree)
			w.Println(stats.String())

			// traverse tree
			output := make(chan internals.ReportTailLine)
			errChan := make(chan error)
			go internals.HashATree(hashSettings.BaseNode, hashSettings.DFS, hashSettings.IgnorePermErrors,
				hashSettings.HashAlgorithm, hashSettings.ExcludeBasename, hashSettings.ExcludeBasenameRegex,
				hashSettings.ExcludeTree, hashSettings.BasenameMode, hashSettings.Workers, output, errChan,
			)

			// read value from evaluation
			digest := make([]byte, 128) // 128 bytes = 1024 bits digest output
			for tailline := range output {
				if tailline.Path == "." {
					copy(digest, tailline.HashValue)
				}
			}

			err, ok := <-errChan
			if ok {
				// TODO errChan does not propagate appropriate exit code
				return 6, jsonOutput, err
			}

			if jsonOutput {
				type jsonResult struct {
					Digest string `json:"digest"`
				}

				data := jsonResult{Digest: hex.EncodeToString(digest)}
				jsonRepr, err := json.Marshal(&data)
				if err != nil {
					return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
				}

				w.Println(string(jsonRepr))
			} else {
				w.Println(hex.EncodeToString(digest))
			}

		} else {

			// NOTE in this case, we don't generate fsstats
			algo, err := internals.HashAlgorithmFromString(hashSettings.HashAlgorithm)
			if err != nil {
				return 8, jsonOutput, err
			}
			hash := algo.Algorithm()
			digest := internals.HashNode(hash, hashSettings.BasenameMode, filepath.Dir(hashSettings.BaseNode), internals.FileData{
				Path:   filepath.Base(hashSettings.BaseNode),
				Type:   internals.DetermineNodeType(fileinfo),
				Size:   uint64(fileinfo.Size()),
				Digest: []byte{},
			})

			if jsonOutput {
				type jsonResult struct {
					Digest string `json:"digest"`
				}

				data := jsonResult{Digest: hex.EncodeToString(digest)}
				jsonRepr, err := json.Marshal(&data)
				if err != nil {
					return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
				}

				w.Println(string(jsonRepr))
			} else {
				w.Println(hex.EncodeToString(digest))
			}
		}

	case diff.cmd.FullCommand():
		diffSettings, err := diff.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}
		jsonOutput = diffSettings.JSONOutput

		if diffSettings.ConfigOutput {
			// config output is printed in JSON independent of diffSettings.JSONOutput
			b, err := json.Marshal(diffSettings)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_CONFIG_JSON, err)
			}
			w.Println(string(b))
			return 0, jsonOutput, nil
		}

		type Identifier struct {
			Digest   string
			BaseName string
		}
		type match []bool
		type matches map[Identifier]match

		// use the first set to determine the set
		diffMatches := make(matches)
		anyFound := make([]bool, len(diffSettings.Targets))
		for t, match := range diffSettings.Targets {
			rep, err := internals.NewReportReader(match.Report)
			if err != nil {
				return 1, jsonOutput, err
			}
			fmt.Fprintf(os.Stderr, "# %s ⇒ %s\n", match.Report, match.BaseNode)
			for {
				tail, err := rep.Iterate()
				if err == io.EOF {
					break
				}
				if err != nil {
					rep.Close()
					return 9, jsonOutput, fmt.Errorf(`failure reading report file '%s' tailline: %s`, match.Report, err)
				}

				// TODO this assumes that paths are canonical and do not end with a folder separator
				if tail.Path == match.BaseNode && (tail.NodeType == 'D' || tail.NodeType == 'L') {
					anyFound[t] = true
				}
				if filepath.Dir(tail.Path) != match.BaseNode {
					continue
				}

				given := Identifier{Digest: string(tail.HashValue), BaseName: filepath.Base(tail.Path)}
				value, ok := diffMatches[given]
				if ok {
					value[t] = true
				} else {
					diffMatches[given] = make([]bool, len(diffSettings.Targets))
					diffMatches[given][t] = true
				}
			}
			rep.Close()
		}

		if diffSettings.JSONOutput {
			type jsonObject struct {
				Basename string   `json:"basename"`
				Digest   string   `json:"digest"`
				OccursIn []string `json:"occurs-in"`
			}
			type jsonResult struct {
				Children []jsonObject `json:"children"`
			}

			data := jsonResult{Children: make([]jsonObject, 0, len(diffMatches))}
			for id, diffMatch := range diffMatches {
				occurences := make([]string, 0, len(diffSettings.Targets))
				for i, matches := range diffMatch {
					if matches {
						occurences = append(occurences, diffSettings.Targets[i].Report)
					}
				}
				data.Children = append(data.Children, jsonObject{
					Basename: id.BaseName,
					Digest:   hex.EncodeToString([]byte(id.Digest)),
					OccursIn: occurences,
				})
			}

			jsonRepr, err := json.Marshal(&data)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
			}
			w.Println(string(jsonRepr))

		} else {
			for i, anyMatch := range anyFound {
				if !anyMatch {
					log.Printf("# not found: '%s' in '%s'\n", diffSettings.Targets[i].Report, diffSettings.Targets[i].BaseNode)
				}
			}

			w.Println("")
			w.Println("# '+' means found, '-' means missing")

			for id, diffMatch := range diffMatches {
				for _, matched := range diffMatch {
					if matched {
						w.Printf("+")
					} else {
						w.Printf("-")
					}
				}
				w.Printfln("\t%s\t%s", hex.EncodeToString([]byte(id.Digest)), id.BaseName)
			}
		}

	case hashAlgos.cmd.FullCommand():
		hashAlgosSettings, err := hashAlgos.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}
		jsonOutput = hashAlgosSettings.JSONOutput

		if hashAlgosSettings.ConfigOutput {
			// config output is printed in JSON independent of hashAlgosSettings.JSONOutput
			b, err := json.Marshal(hashAlgosSettings)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_CONFIG_JSON, err)
			}
			w.Println(string(b))
			return 0, jsonOutput, nil
		}

		type dataSet struct {
			CheckSucceeded bool     `json:"check-result"`
			SupHashAlgos   []string `json:"supported-hash-algorithms"`
		}

		data := dataSet{
			CheckSucceeded: false,
			SupHashAlgos:   internals.SupportedHashAlgorithms(),
		}

		if hashAlgosSettings.CheckSupport != "" {
			for _, h := range internals.SupportedHashAlgorithms() {
				if h == hashAlgosSettings.CheckSupport {
					data.CheckSucceeded = true
				}
			}
		}

		if jsonOutput {
			b, err := json.Marshal(&data)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
			}
			w.Println(string(b))
		} else {
			jsonRepr, err := json.MarshalIndent(&data, "", "  ")
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
			}
			w.Println(string(jsonRepr))
		}

	case version.cmd.FullCommand():
		versionSettings, err := version.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}
		jsonOutput = versionSettings.JSONOutput

		if versionSettings.ConfigOutput {
			// config output is printed in JSON independent of versionSettings.JSONOutput
			b, err := json.Marshal(versionSettings)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_CONFIG_JSON, err)
			}
			w.Println(string(b))
			return 0, jsonOutput, nil
		}

		versionString := fmt.Sprintf("%d.%d.%d", v1.VERSION_MAJOR, v1.VERSION_MINOR, v1.VERSION_PATCH)

		if jsonOutput {
			type jsonResult struct {
				Version string `json:"version"`
			}

			data := jsonResult{Version: versionString}
			b, err := json.Marshal(&data)
			if err != nil {
				return 6, jsonOutput, fmt.Errorf(E_RESULT_JSON, err)
			}
			w.Println(string(b))

		} else {
			w.Println(versionString)
		}

	default:
		return 0, jsonOutput, fmt.Errorf(`unknown subcommand '%s'`, subcommand)
	}

	return 0, jsonOutput, nil
}

func main() {
	// this output stream will be filled with text or JSON (if --json) output
	output := plainOutput{device: os.Stdout}
	// this output stream will be used for status messages
	logOutput := plainOutput{device: os.Stderr}

	exitcode, jsonOutput, err := cli(&output, &logOutput)
	// TODO update design document for the following exit codes
	//   1 → I/O error when reading a file like "file does not exist"
	//   2 → I/O error when writing a file like "could not create file"
	//   3 → I/O error “file exists already” but --overwrite is not specified
	//   4 → I/O error if a file system cycle is detected (currently not checked/triggered)
	//   5 → I/O error if the source file is not UTF-8 encoded (currently not checked/triggered)
	//   6 → I/O error for generic/other cases like "serializing data to JSON failed"
	//   7 → CLI error if some argument is missing or CLI is incorrect
	//   8 → value error if some CLI argument has some invalid content
	//   9 → value error if some report file contains invalid content
	//   10 → command line was invalid, like argument type or missing required argument
	//   11 → internal programming error - please report me! (assertion/invariant failed)

	if err == nil {
		os.Exit(exitcode)
	}

	if jsonOutput {
		type jsonError struct {
			Message  string `json:"error"`
			ExitCode int    `json:"code"`
		}

		jsonData := jsonError{
			Message:  err.Error(),
			ExitCode: exitcode,
		}
		jsonRepr, err := json.Marshal(&jsonData)
		if err != nil {
			// ignore return value intentionally, as we cannot do anything about it
			logOutput.Printfln(`error (exitcode=%d) was thrown: %s`, exitcode, err.Error())
			logOutput.Printfln(`but I failed to create a response in JSON`)
			output.Println(`{"error":"could not encode error message as JSON","exitcode":2}`)
			exitcode = 6
		} else {
			output.Println(string(jsonRepr))
		}
	} else {
		output.Printfln("Error: %s", err.Error())
	}

	os.Exit(exitcode)
}
