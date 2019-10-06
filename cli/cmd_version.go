package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
)

// CLI command parameters
type versionCommand struct {
	ConfigOutput bool
	JSONOutput   bool
	Help         bool
}

// kingpin CLI arguments
type cliversionCommand struct {
	cmd          *kingpin.CmdClause
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

func newCLIversionCommand(app *kingpin.Application) *cliversionCommand {
	c := new(cliversionCommand)
	c.cmd = app.Command("version", "Print implementation version, license and author. Exit code is always 0.")

	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *cliversionCommand) Validate() (*versionCommand, error) {
	// validity checks (check conditions not covered by kingpin)
	// (nothing.)

	// migrate cliversionCommand to versionCommand
	cmd := new(versionCommand)
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// default values
	if envToBool("DUPFILES_JSON") {
		cmd.JSONOutput = true
	}

	return cmd, nil
}
