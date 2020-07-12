package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/meisterluk/dupfiles-go/internals"
	"github.com/spf13/cobra"
)

// DiffCommand defines the CLI command parameters
type DiffCommand struct {
	Nodes        []NodePathPair `json:"nodes"`
	ConfigOutput bool           `json:"config"`
	JSONOutput   bool           `json:"json"`
	Help         bool           `json:"help"`
}

// NodePathPair contain a filesystem node and the report file mentioning it.
// Pairs of these constitute the arguments you need to provide for subcommand diff.
type NodePathPair struct {
	BaseNode string `json:"path"`
	Report   string `json:"report"`
}

// DiffJSONResult is a struct used to serialize JSON output
type DiffJSONResult struct {
	Children []DiffJSONObject `json:"children"`
}

// DiffJSONObject represents one difference match of the diff command
type DiffJSONObject struct {
	Basename string   `json:"basename"`
	Digest   string   `json:"digest"`
	OccursIn []string `json:"occurs-in"`
}

var diffCommand *DiffCommand

var argPairItems []string

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show difference between filesystem nodes",
	Long: `‘dupfiles diff’ allows the user to compute differences between nodes. A node is specified as a pair of (filepath, report) and given two or more of these pairs, the difference between the given filepaths in the respective report file (based on the metadata such as digest, filesize, basename, and directory structure) is computed. The directory is relative to the root of the report file.
For example:

  dupfiles diff etc/ca-certificates laptop_2020-07-03.fsr
                miscellaneous/etc_backup/ca-certificates nas_xmas_backup.fsr

… returns the child nodes of the two mentioned filepaths and their respective state in comparison to the other filepaths.

‘dupfiles diff’ is currently limited and can only compare filepaths non-recursively. Recursive comparison is planned for future releases, but be aware that visualizations of filesystem differences is difficult. No solutions are known, so no guarantees are provided.
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global DiffCommand instance
	// called diffCommand and fill it with admissible parameters to run the diff command.
	// It EITHER succeeds, fill diffCommand appropriately and returns nil.
	// OR returns an error instance and diffCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// consider positional arguments as argPairItems
		for _, arg := range args {
			argPairItems = append(argPairItems, arg)
		}

		// create global DiffCommand instance
		diffCommand = new(DiffCommand)
		diffCommand.Nodes = make([]NodePathPair, 0, 8)
		diffCommand.ConfigOutput = argConfigOutput
		diffCommand.JSONOutput = argJSONOutput
		diffCommand.Help = false

		// validate Nodes
		a := argPairItems
		if len(a) == 0 {
			exitCode = 7
			return fmt.Errorf(`At least two [{filepath} {report}] pairs are required for comparison, found %d`, len(a))
		} else if len(a)%2 != 0 {
			exitCode = 7
			return fmt.Errorf(`[{filepath} {report}] pairs required. Thus I expected an even number of arguments, got %d`, len(a))
		}
		for i := 0; i < len(a); i = i + 2 {
			if a[i] == "" {
				a[i] = "."
			} else if a[i] == "" {
				exitCode = 8
				return fmt.Errorf(`empty report filepath for '%s' found; expected a valid filepath`, a[i])
			}
			for len(a[i]) > 0 && a[i][len(a[i])-1] == filepath.Separator {
				a[i] = a[i][:len(a[i])-1]
			}
			diffCommand.Nodes = append(diffCommand.Nodes, NodePathPair{BaseNode: a[i], Report: a[i+1]})
		}

		// handle environment variables
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			diffCommand.JSONOutput = envJSON
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, versionCommand}
		exitCode, cmdError = diffCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
	argPairItems = make([]string, 0, 8)
	diffCmd.PersistentFlags().StringSliceVar(&argPairItems, `pair-item`, []string{}, `filepath or report item considered for comparison`)
}

// Run executes the CLI command diff on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *DiffCommand) Run(w Output, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	type Identifier struct {
		Digest   string
		BaseName string
	}
	type match []bool
	type matches map[Identifier]match

	// use the first set to determine the entire set
	diffMatches := make(matches)
	anyFound := make([]bool, len(c.Nodes))
	for t, match := range c.Nodes {
		rep, err := internals.NewReportReader(match.Report)
		if err != nil {
			return 1, err
		}
		log.Printfln("# %s ⇒ %s", match.Report, match.BaseNode)
		for {
			tail, _, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				rep.Close()
				return 9, fmt.Errorf(`failure reading report file '%s' tailline: %s`, match.Report, err)
			}

			// TODO this assumes that paths are canonical and do not end with a folder separator
			//   → since filepath information is now ignored, this should be fine again, right?
			if tail.Path == match.BaseNode && (tail.NodeType == 'D' || tail.NodeType == 'L') {
				anyFound[t] = true
			}
			if !strings.HasPrefix(tail.Path, match.BaseNode) || internals.DetermineDepth(tail.Path, rep.Head.Separator)-1 != internals.DetermineDepth(match.BaseNode, rep.Head.Separator) {
				continue
			}

			given := Identifier{Digest: internals.Hash(tail.HashValue).Digest(), BaseName: internals.Base(tail.Path, rep.Head.Separator)}
			value, ok := diffMatches[given]
			if ok {
				value[t] = true
			} else {
				diffMatches[given] = make([]bool, len(c.Nodes))
				diffMatches[given][t] = true
			}
		}
		rep.Close()
	}

	if c.JSONOutput {
		data := DiffJSONResult{Children: make([]DiffJSONObject, 0, len(diffMatches))}
		for id, diffMatch := range diffMatches {
			occurences := make([]string, 0, len(c.Nodes))
			for i, matches := range diffMatch {
				if matches {
					occurences = append(occurences, c.Nodes[i].Report)
				}
			}
			data.Children = append(data.Children, DiffJSONObject{
				Basename: id.BaseName,
				Digest:   hex.EncodeToString([]byte(id.Digest)),
				OccursIn: occurences,
			})
		}

		jsonRepr, err := json.Marshal(&data)
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))

	} else {
		for i, anyMatch := range anyFound {
			if !anyMatch {
				log.Printf("# not found: '%s' in '%s'\n", c.Nodes[i].Report, c.Nodes[i].BaseNode)
			}
		}

		w.Println("")
		w.Println("# '+' means found, '-' means missing")

		for id, diffMatch := range diffMatches {
			for _, matched := range diffMatch {
				if matched {
					w.Printf("+")
				} else {
					w.Printf("-")
				}
			}
			w.Printfln("\t%s\t%s", hex.EncodeToString([]byte(id.Digest)), id.BaseName)
		}
	}

	return 0, nil
}
