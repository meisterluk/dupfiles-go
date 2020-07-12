package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/meisterluk/dupfiles-go/internals"
	"github.com/spf13/cobra"
)

// GenerateCommand defines the CLI command parameters
type GenerateCommand struct {
	BaseNode             string   `json:"basenode"`
	BaseNodeName         string   `json:"basenode-name"`
	Overwrite            bool     `json:"overwrite"`
	Output               string   `json:"output"`
	Continue             bool     `json:"continue"`
	BFS                  bool     `json:"bfs"`
	DFS                  bool     `json:"dfs"`
	IgnorePermErrors     bool     `json:"ignore-perm-errors"`
	HashAlgorithm        string   `json:"hash-algorithm"`
	ExcludeBasename      []string `json:"exclude-basename"`
	ExcludeBasenameRegex []string `json:"exclude-basename-regex"`
	ExcludeTree          []string `json:"exclude-tree"`
	ThreeMode            bool     `json:"three-mode"`
	ContentMode          bool     `json:"content-mode"`
	Workers              int      `json:"workers"`
	ConfigOutput         bool     `json:"config"`
	JSONOutput           bool     `json:"json"`
	Help                 bool     `json:"help"`
}

// ReportJSONResult is a struct used to serialize JSON output
type ReportJSONResult struct {
	Message string `json:"message"`
}

var generateCommand *GenerateCommand

var nonSenseBaseNodeName *regexp.Regexp
var argBaseNode string
var argBaseNodeName string
var argOverwrite bool
var argOutput string
var argDFS bool
var argBFS bool
var argIgnorePermErrors bool
var argHashAlgorithm string
var argExcludeBasename []string
var argExcludeBasenameRegex []string
var argExcludeTree []string
var argThreeMode bool
var argContentMode bool
var argWorkers int

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates a report file",
	Long: `This command creates a report file representing the file system state at the given filesystem path. For example:

	dupfiles generate tests/example_dir

Unless the report filepath is given (i.e. overwritten), the report file has file extension ‘.fsr’.
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global GenerateCommand instance
	// called generateCommand and fill it with admissible parameters to run the generate command.
	// It EITHER succeeds, fill generateCommand appropriately and returns nil.
	// OR returns an error instance and generateCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// validity checks
		if argDFS && argBFS {
			return fmt.Errorf("cannot accept --bfs and --dfs simultaneously")
		}
		if argThreeMode && argContentMode {
			return fmt.Errorf("cannot accept --three-mode and --content-mode simultaneously")
		}
		if len(args) > 0 && argBaseNode == "" {
			argBaseNode = args[0]
		} else if len(args) == 0 && argBaseNode != "" {
			// ignore, argBaseNode is set properly
		} else if len(args) > 0 && argBaseNode != "" {
			return fmt.Errorf(`cannot provide basenode as positional argument and via --basenode; remove --basenode`)
		} else if len(args) == 0 && argBaseNode == "" {
			return fmt.Errorf(`expected one positional argument {basenode}, got none`)
		}

		// create global GenerateCommand instance
		generateCommand = new(GenerateCommand)
		generateCommand.BaseNode = argBaseNode
		generateCommand.BaseNodeName = argBaseNodeName
		generateCommand.Overwrite = argOverwrite
		generateCommand.Output = argOutput
		generateCommand.DFS = argDFS
		generateCommand.BFS = argBFS
		generateCommand.IgnorePermErrors = argIgnorePermErrors
		generateCommand.HashAlgorithm = argHashAlgorithm
		generateCommand.ExcludeBasename = argExcludeBasename
		generateCommand.ExcludeBasenameRegex = argExcludeBasenameRegex
		generateCommand.ExcludeTree = argExcludeTree
		generateCommand.ThreeMode = argThreeMode
		generateCommand.ContentMode = argContentMode
		generateCommand.Workers = argWorkers
		generateCommand.ConfigOutput = argConfigOutput
		generateCommand.JSONOutput = argJSONOutput
		generateCommand.Help = false

		// handle environment variables
		envDFS, errDFS := EnvToBool("DUPFILES_DFS")
		if errDFS == nil {
			generateCommand.DFS = envDFS
			generateCommand.BFS = !envDFS
		}
		envContent, errContent := EnvToBool("DUPFILES_CONTENT_MODE")
		if errContent == nil {
			generateCommand.ContentMode = envContent
			generateCommand.ThreeMode = !envContent
		}
		/// DUPFILES_HASH_ALGORITHM was already handled
		envIPE, errIPE := EnvToBool("DUPFILES_IGNORE_PERM_ERRORS")
		if errIPE == nil {
			generateCommand.IgnorePermErrors = envIPE
		}
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			generateCommand.JSONOutput = envJSON
			// NOTE ↓ ugly hack, to make Execute() return the appropriate value
			argJSONOutput = envJSON
		}
		/// DUPFILES_OUTPUT was already handled
		envOverwrite, errOverwrite := EnvToBool("DUPFILES_OVERWRITE")
		if errOverwrite == nil {
			generateCommand.Overwrite = envOverwrite
		}
		if generateCommand.Workers == 0 {
			if w, ok := EnvToInt("DUPFILES_WORKERS"); ok {
				generateCommand.Workers = w
			} else {
				generateCommand.Workers = CountCPUs()
			}
		}

		// default values
		if generateCommand.BaseNodeName == "" {
			generateCommand.BaseNodeName = filepath.Base(generateCommand.BaseNode)
			if nonSenseBaseNodeName.FindString(generateCommand.BaseNodeName) != "" {
				abs, err := filepath.Abs(generateCommand.BaseNode)
				if err != nil {
					return fmt.Errorf(`failed to determine absolute path of '%s': %s`, generateCommand.BaseNode, err)
				}
				generateCommand.BaseNodeName = filepath.Base(abs)
			}
		}
		if !generateCommand.DFS && !generateCommand.BFS {
			generateCommand.DFS = true
		}
		if !generateCommand.ContentMode && !generateCommand.ThreeMode {
			generateCommand.ThreeMode = true
		}

		if generateCommand.Output == "" {
			generateCommand.Output = generateCommand.BaseNodeName + ".fsr"
		}

		// validity check 2
		if generateCommand.Workers <= 0 {
			return fmt.Errorf("expected --workers to be positive integer, is %d", generateCommand.Workers)
		}

		// sanitize tree paths
		for i := range generateCommand.ExcludeTree {
			generateCommand.ExcludeTree[i] = strings.Trim(generateCommand.ExcludeTree[i], `/\`)
		}

		return nil
	},
	// Run the generate subcommand with generateCommand
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, treeCommand}
		exitCode, cmdError = generateCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	f := generateCmd.PersistentFlags()

	defaultHashAlgo := internals.HashAlgos{}.Default().Instance().Name()

	f.StringVar(&argBaseNode, `basenode`, "", `base node to generate report for`)
	generateCmd.MarkFlagRequired("basenode")
	f.StringVarP(&argBaseNodeName, `basenode-name`, `b`, "", `human-readable base node name in head line`)
	f.BoolVar(&argOverwrite, `overwrite`, false, `if filepath already exists, overwrite it without asking`)
	f.StringVarP(&argOutput, `output`, `o`, EnvOr("DUPFILES_OUTPUT", ""), `target location for report`)
	f.BoolVar(&argDFS, `dfs`, false, `apply depth-first search for file system`)
	f.BoolVar(&argBFS, `bfs`, false, `apply breadth-first search for file system`)
	f.BoolVar(&argIgnorePermErrors, `ignore-perm-errors`, false, `ignore permission errors and continue traversal`)
	f.StringVarP(&argHashAlgorithm, `hash-algorithm`, `a`, EnvOr("DUPFILES_HASH_ALGORITHM", defaultHashAlgo), `hash algorithm to use`)
	f.StringSliceVar(&argExcludeBasename, `exclude-basename`, []string{}, `any file with this particular filename is ignored`)
	f.StringSliceVar(&argExcludeBasenameRegex, `exclude-basename-regex`, []string{}, `exclude files with name matching given POSIX regex`)
	f.StringSliceVar(&argExcludeTree, `exclude-tree`, []string{}, `exclude folder and subfolders of given filepath`)
	f.BoolVar(&argThreeMode, `three-mode`, false, `three mode (thus digests encode type, basename, and content)`)
	f.BoolVar(&argContentMode, `content-mode`, false, `content mode (thus digests match tools like md5sum)`)
	f.IntVar(&argWorkers, `workers`, 0, `number of concurrent traversal units`)

	nonSenseBaseNodeName = regexp.MustCompile(`\.+$`)
}

// Run executes the CLI command report on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *GenerateCommand) Run(w Output, log Output) (int, error) {
	// config output
	if c.ConfigOutput {
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

	// create report
	rep, err := internals.NewReportWriter(c.Output)
	if err != nil {
		return 2, fmt.Errorf(`error creating file '%s': %s`, c.Output, err)
	}
	// NOTE since we create a file descriptor for the output file here already,
	//      we need to exclude it from the walk finding all paths.
	//      We could move file descriptor creation to a later point, but I want
	//      to catch FS writing issues early.
	// TODO doesn't this omit an existing file that will be overwritten?
	c.ExcludeTree = append(c.ExcludeTree, c.Output)

	fullPath, err := filepath.Abs(c.BaseNode)
	if err != nil {
		return 6, err
	}
	err = rep.HeadLine(c.HashAlgorithm, !c.ContentMode, byte(filepath.Separator), c.BaseNodeName, fullPath)
	if err != nil {
		return 6, err
	}

	// walk and write tail lines
	entries := make(chan internals.ReportTailLine)
	errChan := make(chan error)
	go internals.HashATree(
		c.BaseNode, c.DFS, c.IgnorePermErrors, c.HashAlgorithm,
		c.ExcludeBasename, c.ExcludeBasenameRegex, c.ExcludeTree,
		c.ThreeMode, c.Workers, entries, errChan,
	)

	for entry := range entries {
		err = rep.TailLine(entry.HashValue, entry.NodeType, entry.FileSize, entry.Path)
		if err != nil {
			return 2, err
		}
	}

	err, ok := <-errChan
	if ok {
		// TODO proper exit code required
		// TODO JSON output support
		return 6, err
	}

	msg := fmt.Sprintf(`Done. File "%s" written`, c.Output)
	if c.JSONOutput {
		data := ReportJSONResult{Message: msg}
		jsonRepr, err := json.Marshal(&data)
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}

		w.Println(string(jsonRepr))
	} else {
		w.Println(msg)
	}

	return 0, nil
}
