package main

import (
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

// CLI command parameters
type StatsCommand struct {
	Report       string `json:"report"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

// kingpin CLI arguments
type CLIStatsCommand struct {
	cmd          *kingpin.CmdClause
	Report       *string
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

func NewCLIStatsCommand(app *kingpin.Application) *CLIStatsCommand {
	c := new(CLIStatsCommand)
	c.cmd = app.Command("stats", "Prints some statistics about filesystem nodes based on a report.")

	c.Report = c.cmd.Arg("report", "report to consider").Required().String()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *CLIStatsCommand) Validate() (*StatsCommand, error) {
	// validity checks (check conditions not covered by kingpin)
	if *c.Report == "" {
		return nil, fmt.Errorf("One report must be specified")
	}

	// migrate CLIStatsCommand to StatsCommand
	cmd := new(StatsCommand)
	cmd.Report = *c.Report
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// default values
	if envToBool("DUPFILES_JSON") {
		cmd.JSONOutput = true
	}

	return cmd, nil
}
