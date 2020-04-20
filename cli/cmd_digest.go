package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

// DigestCommand defines the CLI command parameters
type DigestCommand struct {
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

// CLIDigestCommand defines the CLI arguments as kingpin requires them
type CLIDigestCommand struct {
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

func NewCLIDigestCommand(app *kingpin.Application) *CLIDigestCommand {
	c := new(CLIDigestCommand)
	c.cmd = app.Command("digest", "Give the digest of an individual node.")

	c.BaseNode = c.cmd.Arg("basenode", "base node to generate report for").Required().String()
	c.DFS = c.cmd.Flag("dfs", "apply depth-first search for file system").Bool()
	c.BFS = c.cmd.Flag("bfs", "apply breadth-first search for file system").Bool()
	c.IgnorePermErrors = c.cmd.Flag("ignore-perm-errors", "ignore permission errors and continue traversal").Bool()
	c.HashAlgorithm = c.cmd.Flag("hash-algorithm", "hash algorithm to use").Default(EnvOr("DUPFILES_HASH_ALGORITHM", string(internals.DefaultHash))).Short('a').String()
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

func (c *CLIDigestCommand) Validate() (*DigestCommand, error) {
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

	// migrate CLIDigestCommand to DigestCommand
	cmd := new(DigestCommand)
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
	if cmd.Workers == 0 {
		if w, ok := EnvToInt("DUPFILES_WORKERS"); ok {
			cmd.Workers = w
		} else {
			cmd.Workers = CountCPUs()
		}
	}

	// default values
	if !cmd.DFS && !cmd.BFS {
		cmd.DFS = true
	}
	if !cmd.EmptyMode && !cmd.BasenameMode {
		cmd.BasenameMode = true
	}

	// validity check 2
	if cmd.Workers <= 0 {
		return nil, fmt.Errorf("expected --workers to be positive integer, is %d", cmd.Workers)
	}

	return cmd, nil
}

// Run executes the CLI command diff on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *DigestCommand) Run(w Output, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	fileinfo, err := os.Stat(c.BaseNode)
	if err != nil {
		return 6, err
	}

	if fileinfo.IsDir() {
		// generate fsstats concurrently
		stats := internals.GenerateStatistics(c.BaseNode, c.IgnorePermErrors, c.ExcludeBasename, c.ExcludeBasenameRegex, c.ExcludeTree)
		w.Println(stats.String())

		// traverse tree
		output := make(chan internals.ReportTailLine)
		errChan := make(chan error)
		go internals.HashATree(c.BaseNode, c.DFS, c.IgnorePermErrors,
			c.HashAlgorithm, c.ExcludeBasename, c.ExcludeBasenameRegex,
			c.ExcludeTree, c.BasenameMode, c.Workers, output, errChan,
		)

		// read value from evaluation
		digest := make([]byte, 128) // 128 bytes = 1024 bits digest output
		for tailline := range output {
			if tailline.Path == "." {
				copy(digest, tailline.HashValue)
			}
		}

		err, ok := <-errChan
		if ok {
			// TODO errChan does not propagate appropriate exit code
			return 6, err
		}

		if c.JSONOutput {
			type jsonResult struct {
				Digest string `json:"digest"`
			}

			data := jsonResult{Digest: hex.EncodeToString(digest)}
			jsonRepr, err := json.Marshal(&data)
			if err != nil {
				return 6, fmt.Errorf(resultJSONErrMsg, err)
			}

			w.Println(string(jsonRepr))
		} else {
			w.Println(hex.EncodeToString(digest))
		}

		return 0, nil

	}

	// NOTE in this case, we don't generate fsstats
	algo, err := internals.HashAlgorithmFromString(c.HashAlgorithm)
	if err != nil {
		return 8, err
	}
	hash := algo.Algorithm()
	digest := internals.HashNode(hash, c.BasenameMode, filepath.Dir(c.BaseNode), internals.FileData{
		Path:   filepath.Base(c.BaseNode),
		Type:   internals.DetermineNodeType(fileinfo),
		Size:   uint64(fileinfo.Size()),
		Digest: []byte{},
	})

	if c.JSONOutput {
		type jsonResult struct {
			Digest string `json:"digest"`
		}

		data := jsonResult{Digest: hex.EncodeToString(digest)}
		jsonRepr, err := json.Marshal(&data)
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}

		w.Println(string(jsonRepr))
	} else {
		w.Println(hex.EncodeToString(digest))
	}

	return 0, nil
}
