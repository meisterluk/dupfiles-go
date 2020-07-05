package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/meisterluk/dupfiles-go/internals"
	"github.com/spf13/cobra"
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
	ThreeMode            bool     `json:"three-mode"`
	ContentMode          bool     `json:"content-mode"`
	Workers              int      `json:"workers"`
	ConfigOutput         bool     `json:"config"`
	JSONOutput           bool     `json:"json"`
	Help                 bool     `json:"help"`
}

// DigestJSONResult is a struct used to serialize JSON output
type DigestJSONResult struct {
	Digest string `json:"digest"`
}

var digestCommand *DigestCommand

// digestCmd represents the digest command
var digestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Give the digest of an individual node",
	Long: `This subcommand allows to retrieve the digest of one or more filesystem nodes.
For example:

	dupfiles digest ./bin/dupfiles

returns one string which is the digest of the specified file.
Of course, this command also works for directories. In this case,
the entire filesystem tree must be built.
For example:

	dupfiles digest ~go/github.com/meisterluk/dupfiles-go
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global DigestCommand instance
	// called digestCommand and fill it with admissible parameters to run the digest command.
	// It EITHER succeeds, fill digestCommand appropriately and returns nil.
	// OR returns an error instance and digestCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// validity checks
		if argBaseNode == "" {
			return fmt.Errorf("basenode must not be empty")
		}
		if argDFS && argBFS {
			return fmt.Errorf("cannot accept --bfs and --dfs simultaneously")
		}
		if argThreeMode && argContentMode {
			return fmt.Errorf("cannot accept --three-mode and --content-mode simultaneously")
		}

		// create global DigestCommand instance
		digestCommand = new(DigestCommand)
		digestCommand.BaseNode = argBaseNode
		digestCommand.DFS = argDFS
		digestCommand.BFS = argBFS
		digestCommand.IgnorePermErrors = argIgnorePermErrors
		digestCommand.HashAlgorithm = argHashAlgorithm
		digestCommand.ExcludeBasename = argExcludeBasename
		digestCommand.ExcludeBasenameRegex = argExcludeBasenameRegex
		digestCommand.ExcludeTree = argExcludeTree
		digestCommand.ThreeMode = argThreeMode
		digestCommand.ContentMode = argContentMode
		digestCommand.Workers = argWorkers
		digestCommand.ConfigOutput = argConfigOutput
		digestCommand.JSONOutput = argJSONOutput
		digestCommand.Help = false

		// handle environment variables
		envDFS, errDFS := EnvToBool("DUPFILES_DFS")
		if errDFS == nil {
			digestCommand.DFS = envDFS
			digestCommand.BFS = !envDFS
		}
		envContent, errContent := EnvToBool("DUPFILES_CONTENT_MODE")
		if errContent == nil {
			digestCommand.ContentMode = envContent
			digestCommand.ThreeMode = !envContent
		}
		/// DUPFILES_HASH_ALGORITHM was already handled
		envIPE, errIPE := EnvToBool("DUPFILES_IGNORE_PERM_ERRORS")
		if errIPE == nil {
			digestCommand.IgnorePermErrors = envIPE
		}
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			digestCommand.JSONOutput = envJSON
			// NOTE ↓ ugly hack, to make Execute() return the appropriate value
			argJSONOutput = envJSON
		}
		/// DUPFILES_OUTPUT was already handled
		if digestCommand.Workers == 0 {
			if w, ok := EnvToInt("DUPFILES_WORKERS"); ok {
				digestCommand.Workers = w
			} else {
				digestCommand.Workers = CountCPUs()
			}
		}

		// default values
		if !digestCommand.DFS && !digestCommand.BFS {
			digestCommand.DFS = true
		}
		if !digestCommand.ContentMode && !digestCommand.ThreeMode {
			digestCommand.ThreeMode = true
		}

		// validity check 2
		if digestCommand.Workers <= 0 {
			return fmt.Errorf("expected --workers to be positive integer, is %d", digestCommand.Workers)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, treeCommand}
		exitCode, cmdError = digestCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(digestCmd)
	f := digestCmd.PersistentFlags()

	defaultHashAlgo := internals.HashAlgos{}.Default().Instance().Name()

	f.StringVar(&argBaseNode, `basenode`, "", `base node to generate report for`)
	digestCmd.MarkFlagRequired("basenode")
	f.BoolVar(&argDFS, `dfs`, false, `apply depth-first search for file system`)
	f.BoolVar(&argBFS, `bfs`, false, `apply breadth-first search for file system`)
	f.BoolVar(&argIgnorePermErrors, `ignore-perm-errors`, false, `ignore permission errors and continue traversal`)
	f.StringVarP(&argHashAlgorithm, `hash-algorithm`, `a`, EnvOr("DUPFILES_HASH_ALGORITHM", defaultHashAlgo), `hash algorithm to use`)
	f.StringSliceVar(&argExcludeBasename, `exclude-basename`, []string{}, `any file with this particular filename is ignored`)
	f.StringSliceVar(&argExcludeBasenameRegex, `exclude-basename-regex`, []string{}, `exclude files with name matching given POSIX regex`)
	f.StringSliceVar(&argExcludeTree, `exclude-tree`, []string{}, `exclude folder and subfolders of given filepath`) // TODO trim any trailing/leading separators
	f.BoolVar(&argThreeMode, `three-mode`, false, `three mode (thus digests encode type, basename, and content)`)
	f.BoolVar(&argContentMode, `content-mode`, false, `content mode (thus digests match tools like md5sum)`)
	f.IntVar(&argWorkers, `workers`, 0, `number of concurrent traversal units`)
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
			c.ExcludeTree, c.ThreeMode, c.Workers, output, errChan,
		)

		// read value from evaluation
		maxHashValueSize := 0
		for h := 0; h < internals.CountHashAlgos; h++ {
			outSize := internals.HashAlgo(h).Instance().OutputSize()
			if outSize > maxHashValueSize {
				maxHashValueSize = outSize
			}
		}
		hashValue := make(internals.Hash, maxHashValueSize)
		for tailline := range output {
			if tailline.Path == "." || tailline.Path == "" {
				copy(hashValue, tailline.HashValue)
			}
		}

		err, ok := <-errChan
		if ok {
			// TODO errChan does not propagate appropriate exit code
			return 6, err
		}

		if c.JSONOutput {
			data := DigestJSONResult{Digest: hashValue.Digest()}
			jsonRepr, err := json.Marshal(&data)
			if err != nil {
				return 6, fmt.Errorf(resultJSONErrMsg, err)
			}

			w.Println(string(jsonRepr))
		} else {
			w.Println(hashValue.Digest())
		}

		return 0, nil

	}

	// NOTE in this case, we don't generate fsstats
	algo, err := internals.HashAlgos{}.FromString(c.HashAlgorithm)
	if err != nil {
		return 8, err
	}
	hashValue := internals.HashNode(algo, c.ThreeMode, filepath.Dir(c.BaseNode), internals.FileData{
		Path:      filepath.Base(c.BaseNode),
		Type:      internals.DetermineNodeType(fileinfo),
		Size:      uint64(fileinfo.Size()),
		HashValue: []byte{},
	})

	if c.JSONOutput {
		data := DigestJSONResult{Digest: hashValue.Digest()}
		jsonRepr, err := json.Marshal(&data)
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}

		w.Println(string(jsonRepr))
	} else {
		w.Println(hashValue.Digest())
	}

	return 0, nil
}
