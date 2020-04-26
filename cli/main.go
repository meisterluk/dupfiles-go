package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/meisterluk/dupfiles-go/internals"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var app *kingpin.Application
var report *CLIReportCommand
var find *CLIFindCommand
var stats *CLIStatsCommand
var digest *CLIDigestCommand
var diff *CLIDiffCommand
var hashAlgos *CLIHashAlgosCommand
var version *CLIVersionCommand

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
	report = NewCLIReportCommand(app)
	find = NewCLIFindCommand(app)
	stats = NewCLIStatsCommand(app)
	digest = NewCLIDigestCommand(app)
	diff = NewCLIDiffCommand(app)
	hashAlgos = NewCLIHashAlgosCommand(app)
	version = NewCLIVersionCommand(app)
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
	output := PlainOutput{device: os.Stdout}
	// this output stream will be used for status messages
	logOutput := PlainOutput{device: os.Stderr}

	exitcode, jsonOutput, err := RunCLI(os.Args[1:], &output, &logOutput)
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
			output.Println(`{"error":"could not encode error message as JSON","exitcode":6}`)
			exitcode = 6
		} else {
			output.Println(string(jsonRepr))
		}
	} else {
		output.Printfln("Error: %s", err.Error())
	}

	os.Exit(exitcode)
}
