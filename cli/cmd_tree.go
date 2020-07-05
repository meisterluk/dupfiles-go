package main

import (
	"encoding/json"
	"fmt"

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
	// TODO introduce --plain (i.e. non-ANSI-colored) output
}

func printTreeNode(w Output, template string, node *TreeNode, pos []int8) {
	// NOTE pos[i] == -1 ⇒ $node is first child
	//      pos[i] == 0 ⇒ $node is some child in the middle
	//      pos[i] == 1 ⇒ $node is last child

	prefix := `  `
	// Box drawing block symbols: ╴─│┌┐└┘├┤┬┴└

	if len(pos) == 0 {
		prefix = ` ─`
		//prefix = `┌`
	} else {
		for _, thisPos := range pos[0 : len(pos)-1] {
			if thisPos == 1 {
				prefix += `  `
			} else {
				prefix += ` │`
			}
		}
		switch pos[len(pos)-1] {
		case -1:
			if len(node.Children) > 0 {
				prefix += `└┬`
			} else {
				prefix += `└─`
			}
		case 0:
			prefix += ` ├`
		case 1:
			prefix += ` └`
		}
	}
	w.Printfln(template, prefix, node.Basename, node.Type, node.Size, node.Digest)
	pos = append(pos, 0)
	for i, child := range node.Children {
		childPos := int8(0)
		if i == 0 {
			childPos = -1
		} else if i == len(node.Children)-1 {
			childPos = 1
		}
		pos[len(pos)-1] = childPos
		printTreeNode(w, template, child, pos)
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
	template := "%s \x1b[97m\x1b[40m%s\x1b[0m  \x1b[37m%b %d \x1b[34m%s\x1b[0m"

	// compute output
	if c.JSONOutput {
		jsonRepr, err := json.MarshalIndent(&data, "", "  ")
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))
	} else {
		// TODO compute appropriate representation
		printTreeNode(w, template, &data, make([]int8, 0, 42))
	}

	return 0, nil
}
