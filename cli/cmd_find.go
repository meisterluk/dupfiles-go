package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/meisterluk/dupfiles-go/internals"
	"github.com/spf13/cobra"
)

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

var findCommand *FindCommand

var argReports []string
var argResultByExitcode bool

// findCmd represents the find command
var findCmd = &cobra.Command{
	Use:   "find",
	Short: "Finds duplicates in report files",
	Long: `This subcommand takes any number of report files and finds equivalent filesystem nodes. Equivalent filesystem nodes will only be returned, if all their respective parents are not equivalent. Hence, only the equivalent nodes closest to root will be reported.
For example:

    dupfiles find example.fsr

Will return the set of equivalent nodes in example.fsr closest to root.
TODO list equivalent nodes for the actual test/example tree
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global FindCommand instance
	// called findCommand and fill it with admissible parameters to run the find command.
	// It EITHER succeeds, fill findCommand appropriately and returns nil.
	// OR returns an error instance and findCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// validity checks
		if len(argReports) == 0 {
			return fmt.Errorf("Expected at least 1 report; 0 are given")
		}

		// create global FindCommand instance
		findCommand = new(FindCommand)
		findCommand.Reports = argReports
		findCommand.Overwrite = argOverwrite
		findCommand.Output = argOutput
		findCommand.ResultByExitcode = argResultByExitcode
		findCommand.Long = argLong
		findCommand.ConfigOutput = argConfigOutput
		findCommand.JSONOutput = argJSONOutput
		findCommand.Help = false

		// handle environment variables
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			findCommand.JSONOutput = envJSON
		}
		/// DUPFILES_OUTPUT was already handled
		envOverwrite, errOverwrite := EnvToBool("DUPFILES_OVERWRITE")
		if errOverwrite == nil {
			findCommand.Overwrite = envOverwrite
		}
		envLong, errLong := EnvToBool("DUPFILES_LONG")
		if errLong == nil {
			findCommand.Long = envLong
		}

		return nil
	},
	// Run the find subcommand with findCommand.
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, versionCommand}
		exitCode, cmdError = findCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(findCmd)
	argReports = make([]string, 0, 8)

	findCmd.PersistentFlags().StringSliceVar(&argReports, `reports`, []string{}, `reports to consider`)
	findCmd.MarkFlagRequired("reports")
	findCmd.PersistentFlags().BoolVar(&argOverwrite, `overwrite`, false, `if filepath already exists, overwrite it without asking`)
	findCmd.PersistentFlags().StringVarP(&argOutput, `output`, `o`, EnvOr("DUPFILES_OUTPUT", "report.dup"), `write duplication results to file, not to stdout`)
	findCmd.PersistentFlags().BoolVar(&argResultByExitcode, `result-by-exitcode`, false, `use exit code 42 on success and if at least one duplicate was found`)
	findCmd.PersistentFlags().BoolVarP(&argLong, `long`, `l`, false, `reread report file to provide more data for each duplicate found`)
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
