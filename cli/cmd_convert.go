package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ConvertCommand defines the CLI command parameters
type ConvertCommand struct {
	Action       string `json:"action"`
	Source       string `json:"src"`
	Destination  string `json:"dst"`
	SubDirPath   string `json:"subdir-path"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

var convertCommand *ConvertCommand

var argAction string
var argSource string
var argDestination string

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert data between different formats",
	Long: `Currently only action ‘subdir’ is supported.
For example:

	dupfiles convert --action=subdir -s superset.fsr -d subset_with_only_etc-apache2.fsr etc/apache2
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global ConvertCommand instance
	// called convertCommand and fill it with admissible parameters to run the convert command.
	// It EITHER succeeds, fill convertCommand appropriately and returns nil.
	// OR returns an error instance and convertCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// create global ConvertCommand instance
		convertCommand = new(ConvertCommand)
		convertCommand.Action = argAction
		convertCommand.Source = argSource
		convertCommand.Destination = argDestination
		convertCommand.ConfigOutput = argConfigOutput
		convertCommand.JSONOutput = argJSONOutput

		// validity checks
		if convertCommand.Action != "subdir" {
			return fmt.Errorf("Only ‘subdir’ action is supported; expected --action='subdir'")
		}
		if len(args) == 0 {
			return fmt.Errorf("Expected directory for subdir reduction as positional argument; got none")
		}
		if len(args) > 1 {
			return fmt.Errorf("Expected one positional argument; got %d", len(args))
		}

		convertCommand.SubDirPath = args[0]

		// handle environment variables
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			convertCommand.JSONOutput = envJSON
			// NOTE ↓ ugly hack, to make Execute() return the appropriate value
			argJSONOutput = envJSON
		}

		return nil
	},
	// Run the stats subcommand with convertCommand
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, convertCommand}
		exitCode, cmdError = convertCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
	convertCmd.PersistentFlags().StringVarP(&argAction, `action`, `a`, ``, `action to apply`)
	convertCmd.PersistentFlags().StringVarP(&argSource, `source`, `s`, ``, `source file`)
	convertCmd.PersistentFlags().StringVarP(&argDestination, `destination`, `d`, ``, `destination file`)
}

// Run executes the CLI command convert on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *ConvertCommand) Run(w, log Output) (int, error) {
	w.Println(`TODO implementation`)
	return 0, nil
}
