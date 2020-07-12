package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// TreeCommand defines the CLI command parameters
type TreeCommand struct {
	ConfigOutput bool `json:"config"`
	JSONOutput   bool `json:"json"`
	Help         bool `json:"help"`
}

// TreeNode represents a node of the tree
type TreeNode struct {
	Digest   string      `json:"digest"`
	Type     byte        `json:"type"`
	Size     int         `json:"size"`
	Basename string      `json:"name"`
	Children []*TreeNode `json:"children"`
}

var treeCommand *TreeCommand
var argIndent string
var argPlain bool

// treeCmd represents the tree command
var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "represent the filesystem tree of a report file as tree",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global TreeCommand instance
	// called treeCommand and fill it with admissible parameters to run the tree command.
	// It EITHER succeeds, fill treeCommand appropriately and returns nil.
	// OR returns an error instance and treeCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// consider report as positional argument
		if len(args) > 1 {
			return fmt.Errorf(`expected only one positional argument; got %s`, strings.Join(args, " "))
		}
		if argReport == "" && len(args) == 0 {
			return fmt.Errorf(`positional argument "report file" required`)
		} else if argReport != "" && len(args) == 0 {
			// ignore, argReport is properly set
		} else if argReport == "" && len(args) > 0 {
			argReport = args[0]
		} else if argReport != "" && len(args) > 0 {
			return fmt.Errorf(`two report files given: "%s" and "%s"; expected only one`, argReport, args[0])
		}

		// create global TreeCommand instance
		treeCommand = new(TreeCommand)
		treeCommand.ConfigOutput = argConfigOutput
		treeCommand.JSONOutput = argJSONOutput
		treeCommand.Help = false

		// handle environment variables
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			treeCommand.JSONOutput = envJSON
			// NOTE ↓ ugly hack, to make Execute() return the appropriate value
			argJSONOutput = envJSON
		}

		return nil
	},
	// Run the tree subcommand with treeCommand
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, treeCommand}
		exitCode, cmdError = treeCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)
	treeCmd.PersistentFlags().StringVar(&argReport, `report`, "", `report to consider`)
	treeCmd.PersistentFlags().StringVar(&argIndent, `indent`, "", `if non-empty, show one basename per line and indent to appropriate depth by repeating this string`)
	treeCmd.PersistentFlags().BoolVar(&argPlain, `plain`, false, `if true, do not use ANSI escape sequences to represent colors`)
}

func printTreeNode(w Output, template string, node *TreeNode, isLast []bool) {
	prefix := ``
	// Box drawing block symbols: ╴─│┌┐└┘├┤┬┴└

	for i, last := range isLast {
		if i == len(isLast)-1 && last {
			prefix += "└"
		} else if i == len(isLast)-1 && !last {
			prefix += "├"
		} else if last {
			prefix += " "
		} else if !last {
			prefix += "│"
		}
	}
	if len(node.Children) > 0 {
		prefix += "┬"
	} else {
		prefix += "─"
	}

	w.Printfln(template, prefix, node.Basename, node.Type, node.Size, node.Digest)
	isLast = append(isLast, false)
	for i, child := range node.Children {
		isLast[len(isLast)-1] = (i == len(node.Children)-1)
		printTreeNode(w, template, child, isLast)
	}
}

// Run executes the CLI command tree on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *TreeCommand) Run(w, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	// fill TreeNode with data
	// TODO fill with actual data from report file
	data := TreeNode{}
	data.Basename = `example.file`
	data.Digest = `a2f70a0015afbdea0d34a5eb550a7547`
	data.Size = 42
	data.Type = 'D'
	data.Children = make([]*TreeNode, 0, 4)

	d1 := TreeNode{
		Basename: `ex`,
		Digest:   `a2f70a0015afbdea0d34a5eb550a7547`,
		Size:     12983,
		Type:     'F',
	}
	data.Children = append(data.Children, &d1)
	data.Children = append(data.Children, &d1)

	d2 := TreeNode{
		Basename: `ex2`,
		Digest:   `a2f70a0015afbdea0d34a5eb550a7547`,
		Size:     12943,
		Type:     'F',
	}
	d2.Children = append(d2.Children, &d1)
	data.Children = append(data.Children, &d2)
	data.Children = append(data.Children, &d1)

	//template := `%s %s  %b %d %s`
	// TODO colorized output only works on linux, take a look at https://github.com/k0kubun/go-ansi
	template := "%s \x1b[97m\x1b[40m%s\x1b[0m\t\x1b[37m%b %d \x1b[34m%s\x1b[0m"

	// compute output
	if c.JSONOutput {
		jsonRepr, err := json.MarshalIndent(&data, "", "  ")
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))
	} else {
		// TODO compute appropriate representation
		printTreeNode(w, template, &data, make([]bool, 0, 42))
	}

	return 0, nil
}
