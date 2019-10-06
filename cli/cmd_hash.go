package main

import (
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

// CLI command parameters
type HashCommand struct {
	BaseNode             string   `json:"basenode"`
	BFS                  bool     `json:"bfs"`
	DFS                  bool     `json:"dfs"`
	IgnorePermErrors     bool     `json:"ignore-perm-errors"`
	HashAlgorithm        string   `json:"hash-algorithm"`
	ExcludeFilename      []string `json:"exclude-filename"`
	ExcludeFilenameRegex []string `json:"exclude-filename-regex"`
	ExcludeTree          []string `json:"exclude-tree"`
	BasenameMode         bool     `json:"basename-mode"`
	EmptyMode            bool     `json:"empty-mode"`
	ConfigOutput         bool     `json:"config"`
	JSONOutput           bool     `json:"json"`
	Help                 bool     `json:"help"`
}

// kingpin CLI arguments
type CLIHashCommand struct {
	cmd                  *kingpin.CmdClause
	BaseNode             *string
	BFS                  *bool
	DFS                  *bool
	IgnorePermErrors     *bool
	HashAlgorithm        *string
	ExcludeFilename      *[]string
	ExcludeFilenameRegex *[]string
	ExcludeTree          *[]string
	BasenameMode         *bool
	EmptyMode            *bool
	ConfigOutput         *bool
	JSONOutput           *bool
	Help                 *bool
}

func NewCLIHashCommand(app *kingpin.Application) *CLIHashCommand {
	c := new(CLIHashCommand)
	c.cmd = app.Command("hash", "Give the hash value of an individual node.")

	c.BaseNode = c.cmd.Arg("basenode", "base node to generate report for").Required().String()
	c.DFS = c.cmd.Flag("dfs", "apply depth-first search for file system").Bool()
	c.BFS = c.cmd.Flag("bfs", "apply breadth-first search for file system").Bool()
	c.IgnorePermErrors = c.cmd.Flag("ignore-perm-errors", "ignore permission errors and continue traversal").Bool()
	c.HashAlgorithm = c.cmd.Flag("hash-algorithm", "hash algorithm to use").Default(envOr("DUPFILES_HASH_ALGORITHM", "fnv-1a-128")).Short('a').String()
	c.ExcludeFilename = c.cmd.Flag("exclude-filename", "any file with this particular filename is ignored").Strings()
	c.ExcludeFilenameRegex = c.cmd.Flag("exclude-filename-regex", "exclude files with name matching given POSIX regex").Strings()
	c.ExcludeTree = c.cmd.Flag("exclude-tree", "exclude folder and subfolders of given filepath").Strings()
	c.BasenameMode = c.cmd.Flag("basename-mode", "basename mode (thus hashes encode structure)").Bool()
	c.EmptyMode = c.cmd.Flag("empty-mode", "empty mode (thus hashes match tools like md5sum)").Bool()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *CLIHashCommand) Validate() (*HashCommand, error) {
	// validity checks (check conditions not covered by kingpin)
	if *c.BaseNode == "" {
		return nil, fmt.Errorf("basenode must not be empty")
	}
	if *c.DFS && *c.BFS {
		return nil, fmt.Errorf("cannot accept --bfs and --dfs simultaneously")
	}
	if envToBool(`DUPFILES_BFS`) && envToBool(`DUPFILES_DFS`) {
		return nil, fmt.Errorf("cannot accept env BFS and DFS simultaneously")
	}
	if *c.BasenameMode && *c.EmptyMode {
		return nil, fmt.Errorf("cannot accept --basename-mode and --empty-mode simultaneously")
	}

	// migrate CLIHashCommand to HashCommand
	cmd := new(HashCommand)
	cmd.ExcludeFilename = make([]string, 0)
	cmd.ExcludeFilenameRegex = make([]string, 0)
	cmd.ExcludeTree = make([]string, 0)

	cmd.BaseNode = *c.BaseNode
	cmd.DFS = *c.DFS
	cmd.BFS = *c.BFS
	cmd.IgnorePermErrors = *c.IgnorePermErrors
	cmd.HashAlgorithm = *c.HashAlgorithm

	copy(cmd.ExcludeFilename, *c.ExcludeFilename)
	copy(cmd.ExcludeFilenameRegex, *c.ExcludeFilenameRegex)
	copy(cmd.ExcludeTree, *c.ExcludeTree)
	cmd.BasenameMode = *c.BasenameMode
	cmd.EmptyMode = *c.EmptyMode
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput
	cmd.Help = false

	// default values
	if !cmd.BFS && !cmd.DFS {
		if envToBool("DUPFILES_BFS") && !envToBool("DUPFILES_DFS") {
			cmd.BFS = true
		} else if !envToBool("DUPFILES_BFS") && envToBool("DUPFILES_DFS") {
			cmd.DFS = true
		} else if !envToBool("DUPFILES_BFS") && !envToBool("DUPFILES_DFS") {
			cmd.BFS = false
			cmd.DFS = true
		}
	}
	if envToBool("DUPFILES_IGNORE_PERM_ERRORS") && !cmd.IgnorePermErrors {
		cmd.IgnorePermErrors = true
	}
	if envToBool("DUPFILES_EMPTY_MODE") && !cmd.BasenameMode {
		cmd.EmptyMode = true
	}
	if envToBool("DUPFILES_JSON") {
		cmd.JSONOutput = true
	}

	return cmd, nil
}
