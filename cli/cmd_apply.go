package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/meisterluk/dupfiles-go/internals"
	"github.com/spf13/cobra"
)

// ApplyCommand defines the CLI command parameters
type ApplyCommand struct {
	Action       string   `json:"action"`
	Args         []string `json:"args"`
	ConfigOutput bool     `json:"config"`
	JSONOutput   bool     `json:"json"`
	Help         bool     `json:"help"`
}

var applyCommand *ApplyCommand
var argAction string
var argArgs []string

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply actions to report files",
	Long: `Currently only action ‘subdir’ is supported.
subdir {A} {B} {C}
  A: report file covering a root directory
  B: subdirectory in root to filter with
  C: subset report file to be written

  dupfiles apply subdir superset.fsr subset_with_only_etc-apache2.fsr etc/apache2
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global ApplyCommand instance
	// called applyCommand and fill it with admissible parameters to run the apply command.
	// It EITHER succeeds, fill applyCommand appropriately and returns nil.
	// OR returns an error instance and applyCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// consider positional arguments
		if argAction != "" {
			args = append([]string{argAction}, args...)
		}

		// create global ApplyCommand instance
		applyCommand = new(ApplyCommand)
		applyCommand.Action = args[0]
		applyCommand.Args = args[1:]
		applyCommand.ConfigOutput = argConfigOutput
		applyCommand.JSONOutput = argJSONOutput

		// validity checks
		if applyCommand.Action != "subdir" {
			exitCode = 8
			return fmt.Errorf("Only ‘subdir’ action is supported; expected --action='subdir'")
		}
		if applyCommand.Action == "subdir" && len(args) != 4 {
			exitCode = 7
			return fmt.Errorf(`subdir expected 3 arguments: {in-report} {out-report} {subdir}; got %d argument(s)`, len(args)-1)
		}

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
	applyCmd.PersistentFlags().StringSliceVar(&argArgs, `args`, []string{}, `arguments for this action`)
}

// Run executes the CLI command apply on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *ApplyCommand) Run(w, log Output) (int, error) {
	switch c.Action {
	case "subdir":
		src := c.Args[0]
		dst := c.Args[1]
		subdir := c.Args[2]

		// if src == dst, use a temporary directory
		if src == dst {
			tmpFile, err := os.CreateTemp(os.TempDir(), "dupfiles-")
			if err != nil {
				return 6, err
			}
			dst = tmpFile.Name()
			defer os.Remove(dst)
		}

		source, err := internals.NewReportReader(src)
		if err != nil {
			return 1, err
		}

		destination, err := internals.NewReportWriter(dst)
		if err != nil {
			return 6, err
		}

		i := 0
		for {
			line, _, err := source.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				return 1, err
			}

			if i == 0 {
				// make sure subdir ends with a separator
				if len(subdir) > 0 && subdir[len(subdir)-1] != source.Head.Separator {
					subdir += string(source.Head.Separator)
				}

				// write head line of destination
				newBasePath := source.Head.BasePath + string(source.Head.Separator) + subdir[0:len(subdir)-1]
				err = destination.HeadLine(
					source.Head.HashAlgorithm, source.Head.ThreeMode,
					source.Head.Separator, source.Head.NodeName, newBasePath,
				)
				if err != nil {
					return 6, err
				}
			}
			i++

			// skip nodes outside subdir
			if !strings.HasPrefix(line.Path, subdir) {
				continue
			}

			err = destination.TailLine(line.HashValue, line.NodeType, line.Size, line.Path[len(subdir):])
			if err != nil {
				return 6, err
			}
		}

		// if a temporary file was written, move it to destination
		actualDst := c.Args[1]
		if dst != actualDst {
			d, err := os.Create(actualDst)
			if err != nil {
				return 2, err
			}
			s, err := os.Open(dst)
			if err != nil {
				return 6, err
			}
			for {
				var buffer [4096]byte
				n, err := s.Read(buffer[:])
				if n == 0 && err == io.EOF {
					break
				}
				if err != nil {
					return 6, err
				}
				_, err = d.Write(buffer[:n])
				if err != nil {
					return 6, err
				}
			}
		}
	}

	return 0, nil
}
