package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/meisterluk/dupfiles-go/internals"
	v1 "github.com/meisterluk/dupfiles-go/v1"
	"gopkg.in/alecthomas/kingpin.v2"
)

var app *kingpin.Application
var report *cliReportCommand
var find *cliFindCommand
var stats *cliStatsCommand
var digest *cliDigestCommand
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
	hashAlgos = newCLIHashAlgosCommand(app)
	version = newCLIVersionCommand(app)
}

func main() {
	subcommand, err := app.Parse(os.Args[1:])

	if err != nil {
		resp := &errorResponse{err.Error(), 1}
		os.Exit(resp.Print())
	}

	switch subcommand {
	case report.cmd.FullCommand():
		reportSettings, err := report.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		// config output
		if reportSettings.ConfigOutput {
			b, err := json.Marshal(reportSettings)
			if err != nil {
				handleError(err.Error(), 2, reportSettings.JSONOutput)
			}

			fmt.Println(string(b))
			os.Exit(0)
		}

		// TODO: implement continue option

		// create report
		// TODO reportSettings.Overwrite is not respected
		rep, err := internals.NewReportWriter(reportSettings.Output)
		if err != nil {
			handleError(err.Error(), 2, reportSettings.JSONOutput)
		}
		// NOTE since we create a file descriptor for the output file here already,
		//      we need to exclude it from the walk finding all paths.
		//      We could move file descriptor creation to a later point, but I want
		//      to catch FS writing issues early.
		reportSettings.ExcludeTree = append(reportSettings.ExcludeTree, reportSettings.Output)

		fullPath, err := filepath.Abs(reportSettings.BaseNode)
		if err != nil {
			handleError(err.Error(), 2, reportSettings.JSONOutput)
		}
		err = rep.HeadLine(reportSettings.HashAlgorithm, !reportSettings.EmptyMode, reportSettings.BaseNodeName, fullPath)
		if err != nil {
			handleError(err.Error(), 2, reportSettings.JSONOutput)
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
				handleError(err.Error(), 2, reportSettings.JSONOutput)
			}
		}

		err, ok := <-errChan
		if ok {
			handleError(err.Error(), 2, reportSettings.JSONOutput)
		}
		os.Exit(0)

	case find.cmd.FullCommand():
		findSettings, err := find.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		if findSettings.ConfigOutput {
			// config output is printed in JSON independent of findSettings.JSONOutput
			b, err := json.Marshal(findSettings)
			if err != nil {
				handleError(err.Error(), 2, findSettings.JSONOutput)
				return
			}
			fmt.Println(string(b))
			return
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
				log.Println(`error:`, err)
			}
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
					jsonDump, err := json.Marshal(entries)
					if err != nil {
						log.Printf(`error marshalling result: %s`, err.Error())
						continue
					}

					os.Stdout.Write(jsonDump)
				}

			} else {
				for entry := range dupEntries {
					//log.Println("<duplicates>")
					out := hex.EncodeToString(entry.Digest) + "\n"
					for _, s := range entry.Set {
						out += `  ` + s.ReportFile + " " + string(filepath.Separator) + " " + s.Path + "\n"
					}
					fmt.Println(out)
					//log.Println("</duplicates>")
				}
			}
		}()

		internals.FindDuplicates(findSettings.Reports, dupEntries, errChan)
		wg.Wait()
		os.Exit(exitCode)

	case stats.cmd.FullCommand():
		statsSettings, err := stats.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		b, err := json.Marshal(statsSettings)
		if err != nil {
			fmt.Println("error:", err)
		}
		fmt.Println(string(b))

	case digest.cmd.FullCommand():
		hashSettings, err := digest.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		if hashSettings.ConfigOutput {
			// config output is printed in JSON independent of hashSettings.JSONOutput
			b, err := json.Marshal(hashSettings)
			if err != nil {
				handleError(err.Error(), 2, hashSettings.JSONOutput)
				return
			}
			fmt.Println(string(b))
			return
		}

		fileinfo, err := os.Stat(hashSettings.BaseNode)
		if err != nil {
			handleError(err.Error(), 1, hashSettings.JSONOutput)
		}

		if fileinfo.IsDir() {
			// generate fsstats concurrently
			stats := internals.GenerateStatistics(hashSettings.BaseNode, hashSettings.IgnorePermErrors, hashSettings.ExcludeBasename, hashSettings.ExcludeBasenameRegex, hashSettings.ExcludeTree)
			log.Println(stats.String())

			// traverse tree
			output := make(chan internals.ReportTailLine)
			errChan := make(chan error)
			go internals.HashATree(hashSettings.BaseNode, hashSettings.DFS, hashSettings.IgnorePermErrors,
				hashSettings.HashAlgorithm, hashSettings.ExcludeBasename, hashSettings.ExcludeBasenameRegex,
				hashSettings.ExcludeTree, hashSettings.BasenameMode, hashSettings.Workers, output, errChan,
			)

			// read value from evaluation
			targetDigest := make([]byte, 128) // 128 bytes = 1024 bits digest output
			for tailline := range output {
				if tailline.Path == "." {
					copy(targetDigest, tailline.HashValue)
				}
			}

			err, ok := <-errChan
			if ok {
				log.Println(err)
			} else {
				fmt.Println(hex.EncodeToString(targetDigest))
			}
		} else {
			// NOTE in this case, we don't generate fsstats
			algo, err := internals.HashAlgorithmFromString(hashSettings.HashAlgorithm)
			if err != nil {
				handleError(err.Error(), 1, hashSettings.JSONOutput)
			}
			hash := algo.Algorithm()
			digest := internals.HashNode(hash, hashSettings.BasenameMode, filepath.Dir(hashSettings.BaseNode), internals.FileData{
				Path:   filepath.Base(hashSettings.BaseNode),
				Type:   internals.DetermineNodeType(fileinfo),
				Size:   uint64(fileinfo.Size()),
				Digest: []byte{},
			})
			fmt.Println(hex.EncodeToString(digest))
		}

	case hashAlgos.cmd.FullCommand():
		hashAlgosSettings, err := hashAlgos.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
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

		b, err := json.Marshal(data)
		if err != nil {
			fmt.Println("error:", err)
		}
		fmt.Println(string(b))

	case version.cmd.FullCommand():
		versionSettings, err := version.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		if versionSettings.ConfigOutput {
			// config output is printed in JSON independent of versionSettings.JSONOutput
			b, err := json.Marshal(versionSettings)
			if err != nil {
				handleError(err.Error(), 2, versionSettings.JSONOutput)
				return
			}
			fmt.Println(string(b))
			return
		}

		versionString := fmt.Sprintf("%d.%d.%d", v1.VERSION_MAJOR, v1.VERSION_MINOR, v1.VERSION_PATCH)

		if !versionSettings.JSONOutput {
			fmt.Println(versionString)

		} else {
			type output struct {
				Version string `json:"version"`
			}

			b, err := json.Marshal(&output{versionString})
			if err != nil {
				handleError(err.Error(), 2, versionSettings.JSONOutput)
				return
			}
			fmt.Println(string(b))
			return
		}

	default:
		kingpin.FatalUsage("unknown command")
	}
}
