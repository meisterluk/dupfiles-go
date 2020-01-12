package main

import (
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

// StatsCommand defines the CLI command parameters
type StatsCommand struct {
	Report       string `json:"report"`
	Long         bool   `json:"long"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

// cliStatsCommand defines the CLI arguments as kingpin requires them
type cliStatsCommand struct {
	cmd          *kingpin.CmdClause
	Report       *string
	Long         *bool
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

func newCLIStatsCommand(app *kingpin.Application) *cliStatsCommand {
	c := new(cliStatsCommand)
	c.cmd = app.Command("stats", "Prints some statistics about filesystem nodes based on a report.")

	c.Report = c.cmd.Arg("report", "report to consider").Required().String()
	c.Long = c.cmd.Flag("long", "compute more features, but takes longer").Bool()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *cliStatsCommand) Validate() (*StatsCommand, error) {
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
	envLong, errLong := envToBool("DUPFILES_LONG")
	if errLong == nil {
		cmd.Long = envLong
	}

	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}
