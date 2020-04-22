package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

// ReportJSONResult is a struct used to serialize JSON output
type ReportJSONResult struct {
	Message string `json:"message"`
}

// CLIReportCommand defines the CLI arguments as kingpin requires them
type CLIReportCommand struct {
	cmd                  *kingpin.CmdClause
	BaseNode             *string
	BaseNodeName         *string
	Overwrite            *bool
	Output               *string
	Continue             *bool
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
}

// NewCLIReportCommand defines the flags/arguments the CLI parser is supposed to understand
func NewCLIReportCommand(app *kingpin.Application) *CLIReportCommand {
	c := new(CLIReportCommand)
	c.cmd = app.Command("report", "Generates a report file.")

	defaultHashAlgo := internals.HashAlgos{}.Default().Instance().Name()

	c.BaseNode = c.cmd.Arg("basenode", "base node to generate report for").Required().String()
	c.BaseNodeName = c.cmd.Flag("basenode-name", "human-readable base node name in head line").Short('b').String()
	c.Overwrite = c.cmd.Flag("overwrite", "if filepath already exists, overwrite it without asking").Bool()
	c.Output = c.cmd.Flag("output", "target location for report").Default(EnvOr("DUPFILES_OUTPUT", "")).Short('o').String()
	c.Continue = c.cmd.Flag("continue", "assume that the output file is incomplete and we continue processing").Short('c').Bool()
	c.DFS = c.cmd.Flag("dfs", "apply depth-first search for file system").Bool()
	c.BFS = c.cmd.Flag("bfs", "apply breadth-first search for file system").Bool()
	c.IgnorePermErrors = c.cmd.Flag("ignore-perm-errors", "ignore permission errors and continue traversal").Bool()
	c.HashAlgorithm = c.cmd.Flag("hash-algorithm", "hash algorithm to use").Default(EnvOr("DUPFILES_HASH_ALGORITHM", defaultHashAlgo)).Short('a').String()
	c.ExcludeBasename = c.cmd.Flag("exclude-basename", "any file with this particular filename is ignored").Strings()
	c.ExcludeBasenameRegex = c.cmd.Flag("exclude-basename-regex", "exclude files with name matching given POSIX regex").Strings()
	c.ExcludeTree = c.cmd.Flag("exclude-tree", "exclude folder and subfolders of given filepath").Strings() // TODO trim any trailing/leading separators
	c.BasenameMode = c.cmd.Flag("basename-mode", "basename mode (thus hashes encode structure)").Bool()
	c.EmptyMode = c.cmd.Flag("empty-mode", "empty mode (thus hashes match tools like md5sum)").Bool()
	c.Workers = c.cmd.Flag("workers", "number of concurrent traversal units").Int()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

// Validate renders all arguments into a ReportCommand or throws an error.
// ReportCommand provides *all* arguments to run a 'report' command.
func (c *CLIReportCommand) Validate() (*ReportCommand, error) {
	// validity checks (check conditions not covered by kingpin)
	if *c.BaseNode == "" {
		return nil, fmt.Errorf("basenode must not be empty")
	}
	if *c.DFS && *c.BFS {
		return nil, fmt.Errorf("cannot accept --bfs and --dfs simultaneously")
	}
	if *c.BasenameMode && *c.EmptyMode {
		return nil, fmt.Errorf("cannot accept --basename-mode and --empty-mode simultaneously")
	}

	// migrate CLIReportCommand to ReportCommand
	cmd := new(ReportCommand)
	cmd.ExcludeBasename = make([]string, 0)
	cmd.ExcludeBasenameRegex = make([]string, 0)
	cmd.ExcludeTree = make([]string, 0)

	cmd.BaseNode = *c.BaseNode
	cmd.BaseNodeName = *c.BaseNodeName
	cmd.Overwrite = *c.Overwrite
	cmd.Output = *c.Output
	cmd.Continue = *c.Continue
	cmd.DFS = *c.DFS
	cmd.BFS = *c.BFS
	cmd.IgnorePermErrors = *c.IgnorePermErrors
	cmd.HashAlgorithm = *c.HashAlgorithm

	copy(cmd.ExcludeBasename, *c.ExcludeBasename)
	copy(cmd.ExcludeBasenameRegex, *c.ExcludeBasenameRegex)
	copy(cmd.ExcludeTree, *c.ExcludeTree)
	cmd.BasenameMode = *c.BasenameMode
	cmd.EmptyMode = *c.EmptyMode
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.Workers = *c.Workers
	cmd.JSONOutput = *c.JSONOutput
	cmd.Help = false

	// handle environment variables
	envDFS, errDFS := EnvToBool("DUPFILES_DFS")
	if errDFS == nil {
		cmd.DFS = envDFS
		cmd.BFS = !envDFS
	}
	envEmpty, errEmpty := EnvToBool("DUPFILES_EMPTY_MODE")
	if errEmpty == nil {
		cmd.EmptyMode = envEmpty
		cmd.BasenameMode = !envEmpty
	}
	/// DUPFILES_HASH_ALGORITHM was already handled
	envIPE, errIPE := EnvToBool("DUPFILES_IGNORE_PERM_ERRORS")
	if errIPE == nil {
		cmd.IgnorePermErrors = envIPE
	}
	envJSON, errJSON := EnvToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}
	/// DUPFILES_OUTPUT was already handled
	envOverwrite, errOverwrite := EnvToBool("DUPFILES_OVERWRITE")
	if errOverwrite == nil {
		cmd.Overwrite = envOverwrite
	}
	if cmd.Workers == 0 {
		if w, ok := EnvToInt("DUPFILES_WORKERS"); ok {
			cmd.Workers = w
		} else {
			cmd.Workers = CountCPUs()
		}
	}

	// default values
	if cmd.BaseNodeName == "" {
		cmd.BaseNodeName = filepath.Base(cmd.BaseNode)
	}
	if !cmd.DFS && !cmd.BFS {
		cmd.DFS = true
	}
	if !cmd.EmptyMode && !cmd.BasenameMode {
		cmd.BasenameMode = true
	}

	if cmd.Output == "" {
		if cmd.BaseNodeName == "." || cmd.BaseNodeName == "" {
			cmd.Output = "report.fsr"
		} else {
			cmd.Output = cmd.BaseNodeName + ".fsr"
		}
	}

	// validity check 2
	if cmd.Workers <= 0 {
		return nil, fmt.Errorf("expected --workers to be positive integer, is %d", cmd.Workers)
	}

	return cmd, nil
}

// ReportCommand defines the CLI command parameters
type ReportCommand struct {
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
	BasenameMode         bool     `json:"basename-mode"`
	EmptyMode            bool     `json:"empty-mode"`
	Workers              int      `json:"workers"`
	ConfigOutput         bool     `json:"config"`
	JSONOutput           bool     `json:"json"`
	Help                 bool     `json:"help"`
}

// Run executes the CLI command report on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *ReportCommand) Run(w Output, log Output) (int, error) {
	// config output
	if c.ConfigOutput {
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}

		w.Println(string(b))
		return 0, nil
	}

	// TODO: implement continue option

	// consider c.Overwrite
	_, err := os.Stat(c.Output)
	if err == nil && !c.Overwrite {
		return 3, fmt.Errorf(existsErrMsg, c.Output)
	}

	// create report
	rep, err := internals.NewReportWriter(c.Output)
	if err != nil {
		return 2, fmt.Errorf(`error writing file '%s': %s`, c.Output, err)
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
	err = rep.HeadLine(c.HashAlgorithm, !c.EmptyMode, byte(filepath.Separator), c.BaseNodeName, fullPath)
	if err != nil {
		return 6, err
	}

	// walk and write tail lines
	entries := make(chan internals.ReportTailLine)
	errChan := make(chan error)
	go internals.HashATree(
		c.BaseNode, c.DFS, c.IgnorePermErrors, c.HashAlgorithm,
		c.ExcludeBasename, c.ExcludeBasenameRegex, c.ExcludeTree,
		c.BasenameMode, c.Workers, entries, errChan,
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
