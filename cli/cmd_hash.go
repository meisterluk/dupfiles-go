package main

import (
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

// HashCommand defines the CLI command parameters
type HashCommand struct {
	BaseNode             string   `json:"basenode"`
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

// cliHashCommand defines the CLI arguments as kingpin requires them
type cliHashCommand struct {
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
}

func newCLIHashCommand(app *kingpin.Application) *cliHashCommand {
	c := new(cliHashCommand)
	c.cmd = app.Command("hash", "Give the hash value of an individual node.")

	c.BaseNode = c.cmd.Arg("basenode", "base node to generate report for").Required().String()
	c.DFS = c.cmd.Flag("dfs", "apply depth-first search for file system").Bool()
	c.BFS = c.cmd.Flag("bfs", "apply breadth-first search for file system").Bool()
	c.IgnorePermErrors = c.cmd.Flag("ignore-perm-errors", "ignore permission errors and continue traversal").Bool()
	c.HashAlgorithm = c.cmd.Flag("hash-algorithm", "hash algorithm to use").Default(envOr("DUPFILES_HASH_ALGORITHM", "fnv-1a-128")).Short('a').String()
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

func (c *cliHashCommand) Validate() (*HashCommand, error) {
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

	// migrate CLIHashCommand to HashCommand
	cmd := new(HashCommand)
	cmd.ExcludeBasename = make([]string, 0)
	cmd.ExcludeBasenameRegex = make([]string, 0)
	cmd.ExcludeTree = make([]string, 0)

	cmd.BaseNode = *c.BaseNode
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
	envDFS, errDFS := envToBool("DUPFILES_DFS")
	if errDFS == nil {
		cmd.DFS = envDFS
		cmd.BFS = !envDFS
	}
	envEmpty, errEmpty := envToBool("DUPFILES_EMPTY_MODE")
	if errEmpty == nil {
		cmd.EmptyMode = envEmpty
		cmd.BasenameMode = !envEmpty
	}
	/// DUPFILES_HASH_ALGORITHM was already handled
	envIPE, errIPE := envToBool("DUPFILES_IGNORE_PERM_ERRORS")
	if errIPE == nil {
		cmd.IgnorePermErrors = envIPE
	}
	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}
	/// DUPFILES_OUTPUT was already handled
	if cmd.Workers == 0 {
		if w, ok := envToInt("DUPFILES_WORKERS"); ok {
			cmd.Workers = w
		} else {
			cmd.Workers = countCPUs()
		}
	}

	// validity check 2
	if cmd.Workers <= 0 {
		return nil, fmt.Errorf("expected --workers to be positive integer, is %d", cmd.Workers)
	}

	return cmd, nil
}
