package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/meisterluk/dupfiles-go/internals"
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

func init() {
	app = kingpin.New("dupfiles", "Determine duplicate files and folders.")
	app.Version("1.0.0").Author("meisterluk")
	app.HelpFlag.Short('h')

	// if --json, show help as JSON
	if internals.Contains(os.Args[1:], "--json") {
		app.UsageTemplate(usageTemplate)
	} else {
		app.UsageTemplate(kingpin.CompactUsageTemplate)
	}

	// initialize subcommand variables
	report = newCLIReportCommand(app)
	find = newCLIFindCommand(app)
	stats = newCLIStatsCommand(app)
	digest = newCLIDigestCommand(app)
	diff = newCLIDiffCommand(app)
	hashAlgos = newCLIHashAlgosCommand(app)
	version = newCLIVersionCommand(app)
}

// RunCLI executes the command line given in args.
// It basically dispatches to a subcommand.
// It writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func RunCLI(args []string, w Output, log Output) (int, bool, error) {
	var exitCode int
	var jsonOutput bool
	var err error

	// <profiling> TODO
	/*f, err := os.Create("cpu.prof")
	if err != nil {
		w.Println(err)
		return 200
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()*/
	// </profiling>

	subcommand, err := app.Parse(args)
	if err != nil {
		return 1, jsonOutput, err
	}

	switch subcommand {
	case report.cmd.FullCommand():
		var reportSettings *ReportCommand
		reportSettings, err = report.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		jsonOutput = reportSettings.JSONOutput
		exitCode, err = reportSettings.Run(w, log)

	case find.cmd.FullCommand():
		var findSettings *FindCommand
		findSettings, err = find.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		jsonOutput = findSettings.JSONOutput
		exitCode, err = findSettings.Run(w, log)

	case stats.cmd.FullCommand():
		var statsSettings *StatsCommand
		statsSettings, err = stats.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		jsonOutput = statsSettings.JSONOutput
		exitCode, err = statsSettings.Run(w, log)

	case digest.cmd.FullCommand():
		var digestSettings *DigestCommand
		digestSettings, err := digest.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		jsonOutput = digestSettings.JSONOutput
		exitCode, err = digestSettings.Run(w, log)

	case diff.cmd.FullCommand():
		var diffSettings *DiffCommand
		diffSettings, err := diff.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		jsonOutput = diffSettings.JSONOutput
		exitCode, err = diffSettings.Run(w, log)

	case hashAlgos.cmd.FullCommand():
		var hashAlgosSettings *HashAlgosCommand
		hashAlgosSettings, err := hashAlgos.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		jsonOutput = hashAlgosSettings.JSONOutput
		exitCode, err = hashAlgosSettings.Run(w, log)

	case version.cmd.FullCommand():
		var versionSettings *VersionCommand
		versionSettings, err := version.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		jsonOutput = versionSettings.JSONOutput
		exitCode, err = versionSettings.Run(w, log)

	default:
		exitCode = 8
		err = fmt.Errorf(`unknown subcommand '%s'`, subcommand)
	}

	return exitCode, jsonOutput, err
}

func main() {
	// this output stream will be filled with text or JSON (if --json) output
	output := plainOutput{device: os.Stdout}
	// this output stream will be used for status messages
	logOutput := plainOutput{device: os.Stderr}

	exitcode, jsonOutput, err := RunCLI(os.Args[1:], &output, &logOutput)
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
	// TODO verify that all input files are properly UTF-8 encoded ⇒ output if properly UTF-8 encoded

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
