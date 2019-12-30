package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

var app *kingpin.Application
var report *cliReportCommand
var find *cliFindCommand
var stats *cliStatsCommand
var hash *cliHashCommand
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
	hash = newCLIHashCommand(app)
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
		rep, err := internals.NewReportWriter(reportSettings.Output)
		if err != nil {
			handleError(err.Error(), 2, reportSettings.JSONOutput)
		}

		err = rep.HeadLine(reportSettings.HashAlgorithm, !reportSettings.EmptyMode, reportSettings.BaseNodeName, reportSettings.BaseNode)
		if err != nil {
			handleError(err.Error(), 2, reportSettings.JSONOutput)
		}

		// walk and write tail lines
		// TODO: is this code goroutine-safe?
		var wg sync.WaitGroup
		pathChan := make(chan string, reportSettings.Workers)
		stop := false
		var anyError error

		// TODO start reportSettings.Workers workers receiving
		//   * paths, consume them, generate hashes

		wg.Add(reportSettings.Workers)
		for w := 0; w < reportSettings.Workers; w++ {
			go func() {
				// fetch Hash instance
				hash, err := internals.HashForHashAlgo(reportSettings.HashAlgorithm)
				if err != nil {
					anyError = err
					wg.Done()
					return
				}
				// receive paths from channels
				for path := range pathChan {
					digest, nodeType, fileSize, err := internals.HashOneNonDirectory(path, hash, !reportSettings.EmptyMode)
					if err != nil {
						anyError = err
						wg.Done()
						return
					}
					err = rep.TailLine(digest, nodeType, fileSize, path)
					if err != nil {
						anyError = err
						wg.Done()
						return
					}
					if stop {
						break
					}
				}
				wg.Done()
			}()
		}

		err = internals.Walk(
			reportSettings.BaseNode,
			reportSettings.BFS,
			reportSettings.IgnorePermErrors,
			reportSettings.ExcludeBasename,
			reportSettings.ExcludeBasenameRegex,
			reportSettings.ExcludeTree,
			pathChan,
		)
		wg.Wait()
		if anyError != nil {
			handleError(anyError.Error(), 2, reportSettings.JSONOutput)
		}
		os.Exit(0)

	case find.cmd.FullCommand():
		findSettings, err := find.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		b, err := json.Marshal(findSettings)
		if err != nil {
			fmt.Println("error:", err)
		}
		fmt.Println(string(b))

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

	case hash.cmd.FullCommand():
		hashSettings, err := hash.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		fileinfo, err := os.Stat(hashSettings.BaseNode)
		if err != nil {
			handleError(err.Error(), 1, hashSettings.JSONOutput)
		}

		if fileinfo.IsDir() {
			// generate fsstats concurrently
			statsChan := make(chan internals.Statistics)
			go internals.GenerateStatistics(hashSettings.BaseNode, hashSettings.IgnorePermErrors, hashSettings.ExcludeBasename, hashSettings.ExcludeBasenameRegex, hashSettings.ExcludeTree, statsChan)

			// pick hash instance
			hash, err := internals.HashForHashAlgo(hashSettings.HashAlgorithm)
			if err != nil {
				handleError(err.Error(), 1, hashSettings.JSONOutput)
			}

			stats := <-statsChan
			log.Println(stats.String())

			/*hashMe(hashSettings.BaseNode, hashSettings.BFS, hashSettings.DFS, hashSettings.IgnorePermErrors, hashSettings.HashAlgorithm, hashSettings.)


			cmd                  *kingpin.CmdClause
			BaseNode             *string
			BFS                  *bool
			DFS                  *bool
			IgnorePermErrors     *bool
			HashAlgorithm        *string
			ExcludeBasename      *[]string
			ExcludeBasenameRegex *[]string
			ExcludeTree          *[]string
			BasenameMode         *bool
			EmptyMode            *bool
			Workers              *int
			ConfigOutput         *bool
			JSONOutput           *bool
			Help                 *bool



			b, err := json.Marshal(hashSettings)
			if err != nil {
				handleError(err.Error(), 2, hashSettings.JSONOutput)
				return
			}
			fmt.Println(string(b))*/

			hash.ReadFile(hashSettings.BaseNode)
			fmt.Println(hash.HexDigest())
		} else {
			// NOTE in this case, we don't generate fsstats
			hash, err := internals.HashForHashAlgo(hashSettings.HashAlgorithm)
			if err != nil {
				handleError(err.Error(), 1, hashSettings.JSONOutput)
			}
			hash.ReadFile(hashSettings.BaseNode)
			fmt.Println(hash.HexDigest())
		}

	case hashAlgos.cmd.FullCommand():
		hashAlgosSettings, err := hashAlgos.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		b, err := json.Marshal(hashAlgosSettings)
		if err != nil {
			fmt.Println("error:", err)
		}
		fmt.Println(string(b))

	case version.cmd.FullCommand():
		versionSettings, err := version.Validate()
		if err != nil {
			kingpin.FatalUsage(err.Error())
		}

		b, err := json.Marshal(versionSettings)
		if err != nil {
			fmt.Println("error:", err)
		}
		fmt.Println(string(b))

	default:
		kingpin.FatalUsage("unknown command")
	}
}
