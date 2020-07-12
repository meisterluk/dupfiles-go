package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meisterluk/dupfiles-go/internals"
	"github.com/spf13/cobra"
)

// TreeCommand defines the CLI command parameters
type TreeCommand struct {
	Report       string `json:"report"`
	Indent       string `json:"indent"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

var treeCommand *TreeCommand
var argIndent string

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
			exitCode = 7
			return fmt.Errorf(`expected only one positional argument; got %s`, strings.Join(args, " "))
		}
		if argReport == "" && len(args) == 0 {
			exitCode = 7
			return fmt.Errorf(`positional argument "report file" required`)
		} else if argReport != "" && len(args) == 0 {
			// ignore, argReport is properly set
		} else if argReport == "" && len(args) > 0 {
			argReport = args[0]
		} else if argReport != "" && len(args) > 0 {
			exitCode = 7
			return fmt.Errorf(`two report files given: "%s" and "%s"; expected only one`, argReport, args[0])
		}

		// create global TreeCommand instance
		treeCommand = new(TreeCommand)
		treeCommand.Report = argReport
		treeCommand.Indent = argIndent
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
}

// PrintTreeNode prints the tree established by TreeNode recursively to w
func PrintTreeNode(w Output, template string, node *internals.TreeNode, isLast []bool) {
	prefix := ``
	basename := node.Basename
	if basename == "" {
		basename = "."
	}

	// Box drawing block symbols: ╴─│┌┐└┘├┤┬┴└
	for i, last := range isLast {
		if i == len(isLast)-1 && last {
			prefix += " └"
		} else if i == len(isLast)-1 && !last {
			prefix += " ├"
		} else if last {
			prefix += "  "
		} else if !last {
			prefix += " │"
		}
	}
	if len(node.Children) > 0 {
		prefix += "─┬"
	} else {
		prefix += "──"
	}

	w.Printfln(template, prefix, basename, node.Type, node.Size, node.Digest)
	isLast = append(isLast, false)
	for i, child := range node.Children {
		isLast[len(isLast)-1] = (i == len(node.Children)-1)
		PrintTreeNode(w, template, child, isLast)
	}
}

// PrintTreeNodeWithIndent prints the tree established by TreeNode recursively to w
// using the $indent prefix string
func PrintTreeNodeWithIndent(w Output, template string, node *internals.TreeNode, depth int, indent string) {
	prefix := strings.Repeat(indent, depth)
	basename := node.Basename
	if basename == "" {
		basename = "."
	}

	w.Printfln(template, prefix, basename, node.Type, node.Size, node.Digest)
	for _, child := range node.Children {
		PrintTreeNodeWithIndent(w, template, child, depth+1, indent)
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

	// get tree from report file
	data, err := internals.TreeFromReport(c.Report)
	if err != nil {
		return 1, err
	}

	// compute output
	template := `%s %s  %c %d %s`
	// template = "%s \x1b[97m\x1b[40m%s\x1b[0m\t\x1b[37m%b %d \x1b[34m%s\x1b[0m"
	if c.JSONOutput {
		jsonRepr, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))
	} else {
		// TODO compute appropriate representation
		if c.Indent == "" {
			PrintTreeNode(w, template, data, make([]bool, 0, 42))
		} else {
			PrintTreeNodeWithIndent(w, template, data, 0, c.Indent)
		}
	}

	return 0, nil
}
