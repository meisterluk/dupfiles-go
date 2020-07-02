package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

// FindJSONResult is a struct used to serialize JSON output
type FindJSONResult struct {
	ReportFile string `json:"report"`
	Path       string `json:"path"`
	Digest     string `json:"digest,omitempty"`
	NodeType   string `json:"type,omitempty"`
	FileSize   uint64 `json:"size,omitempty"`
	LineNo     uint64 `json:"line-number,omitempty"`
	ByteNo     uint64 `json:"byte-offset,omitempty"`
}

// CLIFindCommand defined the CLI arguments as kingpin requires them
type CLIFindCommand struct {
	cmd              *kingpin.CmdClause
	Reports          *[]string
	Overwrite        *bool
	Output           *string
	ResultByExitcode *bool
	Long             *bool
	ConfigOutput     *bool
	JSONOutput       *bool
}

// NewCLIFindCommand defines the flags/arguments the CLI parser is supposed to understand
func NewCLIFindCommand(app *kingpin.Application) *CLIFindCommand {
	c := new(CLIFindCommand)
	c.cmd = app.Command("find", "Finds differences in report files.")

	c.Reports = c.cmd.Arg("reports", "reports to consider").Required().Strings()
	c.Overwrite = c.cmd.Flag("overwrite", "if filepath already exists, overwrite it without asking").Bool()
	c.Output = c.cmd.Flag("output", "write duplication results to file, not to stdout").Short('o').Default(EnvOr("DUPFILES_OUTPUT", "report.dup")).String()
	c.ResultByExitcode = c.cmd.Flag("result-by-exitcode", "use exit code 42 on success and if at least one duplicate was found").Bool()
	c.Long = c.cmd.Flag("long", "reread report file to provide more data for each duplicate found").Short('l').Bool()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

// Validate renders all arguments into a FindCommand or throws an error.
// FindCommand provides *all* arguments to run a 'find' command.
func (c *CLIFindCommand) Validate() (*FindCommand, error) {
	// validity checks (check conditions which are not covered by kingpin)
	if len(*c.Reports) == 0 {
		return nil, fmt.Errorf("At least one report is required")
	}

	// migrate CLIFindCommand to FindCommand
	cmd := new(FindCommand)
	cmd.Reports = make([]string, len(*c.Reports))
	copy(cmd.Reports, *c.Reports)
	cmd.Overwrite = *c.Overwrite
	cmd.Output = *c.Output
	cmd.ResultByExitcode = *c.ResultByExitcode
	cmd.Long = *c.Long
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// handle environment variables
	envJSON, errJSON := EnvToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}
	/// DUPFILES_OUTPUT was already handled
	envOverwrite, errOverwrite := EnvToBool("DUPFILES_OVERWRITE")
	if errOverwrite == nil {
		cmd.Overwrite = envOverwrite
	}
	envLong, errLong := EnvToBool("DUPFILES_LONG")
	if errLong == nil {
		cmd.Long = envLong
	}

	return cmd, nil
}

// FindCommand defines the CLI command parameters
type FindCommand struct {
	Reports          []string `json:"reports"`
	Overwrite        bool     `json:"overwrite"`
	Output           string   `json:"output"`
	ResultByExitcode bool     `json:"result-by-exitcode"`
	Long             bool     `json:"long"`
	ConfigOutput     bool     `json:"config"`
	JSONOutput       bool     `json:"json"`
	Help             bool     `json:"help"`
}

// Run executes the CLI command find on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *FindCommand) Run(w Output, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}

		w.Println(string(b))
		return 0, nil
	}

	// consider c.Overwrite
	_, err := os.Stat(c.Output)
	if err == nil && !c.Overwrite {
		return 3, fmt.Errorf(existsErrMsg, c.Output)
	}

	errChan := make(chan error)
	dupEntries := make(chan internals.DuplicateSet)

	// TODO log number of duplicates
	// TODO log number of duplicate sets
	// TODO --result-by-exitcode

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

		if c.JSONOutput {
			w.Println(`{`)

			w.Println(`  "duplicates": [`)
			var previousJSON string
			for entry := range dupEntries {
				// prepare data structure
				entries := make([]FindJSONResult, 0, len(entry.Set))
				for _, equiv := range entry.Set {
					entries = append(entries, FindJSONResult{
						ReportFile: equiv.ReportFile,
						Path:       equiv.Path,
					})
				}

				// TODO reread file if --long

				// marshal to JSON
				jsonDump, err := json.Marshal(&entries)
				if err != nil {
					log.Printfln(`error marshalling result: %s`, err.Error())
					// TODO? return 6, fmt.Errorf(resultJSONErrMsg, err)
					continue
				}

				if previousJSON != "" {
					w.Println(string(previousJSON) + ",")
				}
				// NOTE previousJSON exists because JSON does not allow trailing commas
				//   in arrays, e.g. `[{}, {}, {},]` is invalid. Thus we need to make sure
				//   the final object is printed without comma.
				previousJSON = string(jsonDump)
			}
			w.Println(string(previousJSON))
			w.Println(`  ]`)
			w.Println(`}`)

		} else {
			for entry := range dupEntries {
				out := internals.Hash(entry.HashValue).Digest() + "\n"
				for _, s := range entry.Set {
					out += `  ` + s.ReportFile + "\t→ " + s.Path + "\n"
				}
				w.Println(out)
			}
		}
	}()

	internals.FindDuplicates(c.Reports, dupEntries, errChan, c.Long)
	wg.Wait()

	// TODO: print debug.GCStats ?
	// TODO exitCode requires better feedback from errChan
	return 0, nil
}
