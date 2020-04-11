package main

import (
	"fmt"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
)

// targetPair contains the basenode and its associated report file.
// Pairs of these constitute the arguments you need to provide for subcommand diff.
type targetPair struct {
	BaseNode string
	Report   string
}

type targetPairs []targetPair

func (t *targetPairs) Set(value string) error {
	if value == "" {
		return fmt.Errorf("'%s' is not a valid base node or report file", value)
	}
	// append to existing
	if len(*t) == 0 || ([]targetPair)(*t)[len(*t)-1].Report != "" {
		*t = append(*t, targetPair{BaseNode: value})
		return nil
	}
	// create new entry
	([]targetPair)(*t)[len(*t)-1].Report = value
	return nil
}

func (t *targetPairs) String() string {
	out := "targetPairs{"
	for _, target := range *t {
		out += fmt.Sprintf(`%s: %s, `, target.BaseNode, target.Report)
	}
	return out + "}"
}

func (t *targetPairs) IsCumulative() bool {
	return true
}

func parseTargets(s *kingpin.ArgClause) *[]targetPair {
	target := new([]targetPair)
	s.SetValue((*targetPairs)(target))
	return target
}

// DiffCommand defines the CLI command parameters
type DiffCommand struct {
	Targets      []targetPair
	ConfigOutput bool
	JSONOutput   bool
	Help         bool
}

// cliDiffCommand defines the CLI arguments as kingpin requires them
type cliDiffCommand struct {
	cmd          *kingpin.CmdClause
	Targets      *[]targetPair
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

func newCLIDiffCommand(app *kingpin.Application) *cliDiffCommand {
	c := new(cliDiffCommand)
	c.cmd = app.Command("diff", "Show difference between node children in two or more report files.")

	c.Targets = parseTargets(c.cmd.Arg("targets", "two or more [{base node} {report}] pairs to consider"))
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *cliDiffCommand) Validate() (*DiffCommand, error) {
	// migrate cliDiffCommand to DiffCommand
	cmd := new(DiffCommand)
	cmd.Targets = make([]targetPair, 0, 8)
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// validate targets
	for _, target := range *c.Targets {
		if target.Report == "" {
			return cmd, fmt.Errorf(`base node '%s' needs the report file path it occurs in`, target.BaseNode)
		}
		for len(target.BaseNode) > 0 && target.BaseNode[len(target.BaseNode)-1] == filepath.Separator {
			target.BaseNode = target.BaseNode[:len(target.BaseNode)-1]
		}
		cmd.Targets = append(cmd.Targets, target)
	}
	if len(cmd.Targets) < 2 {
		return cmd, fmt.Errorf(`At least two [{base node} {report}] pairs are required for comparison, found %d`, len(cmd.Targets))
	}

	// handle environment variables
	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}
