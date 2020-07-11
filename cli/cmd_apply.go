package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ApplyCommand defines the CLI command parameters
type ApplyCommand struct {
	Action       string `json:"action"`
	Source       string `json:"src"`
	Destination  string `json:"dst"`
	SubDirPath   string `json:"subdir-path"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

var applyCommand *ApplyCommand

var argAction string
var argSource string
var argDestination string

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply actions to report files",
	Long: `Currently only action ‘subdir’ is supported.
subdir takes a report file (covering a root directory) and returns a subset report file (covering a subdirectory)

	dupfiles apply --action=subdir -s superset.fsr -d subset_with_only_etc-apache2.fsr etc/apache2
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global ApplyCommand instance
	// called applyCommand and fill it with admissible parameters to run the apply command.
	// It EITHER succeeds, fill applyCommand appropriately and returns nil.
	// OR returns an error instance and applyCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// create global ApplyCommand instance
		applyCommand = new(ApplyCommand)
		applyCommand.Action = argAction
		applyCommand.Source = argSource
		applyCommand.Destination = argDestination
		applyCommand.ConfigOutput = argConfigOutput
		applyCommand.JSONOutput = argJSONOutput

		// validity checks
		if applyCommand.Action != "subdir" {
			return fmt.Errorf("Only ‘subdir’ action is supported; expected --action='subdir'")
		}
		if len(args) == 0 {
			return fmt.Errorf("Expected directory for subdir reduction as positional argument; got none")
		}
		if len(args) > 1 {
			return fmt.Errorf("Expected one positional argument; got %d", len(args))
		}

		applyCommand.SubDirPath = args[0]

		// handle environment variables
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			applyCommand.JSONOutput = envJSON
			// NOTE ↓ ugly hack, to make Execute() return the appropriate value
			argJSONOutput = envJSON
		}

		return nil
	},
	// Run the stats subcommand with applyCommand
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, applyCommand}
		exitCode, cmdError = applyCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.PersistentFlags().StringVarP(&argAction, `action`, `a`, ``, `action to apply`)
	applyCmd.PersistentFlags().StringVarP(&argSource, `source`, `s`, ``, `source file`)
	applyCmd.PersistentFlags().StringVarP(&argDestination, `destination`, `d`, ``, `destination file`)
}

// Run executes the CLI command apply on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *ApplyCommand) Run(w, log Output) (int, error) {
	w.Println(`TODO implementation`)
	return 0, nil
}
