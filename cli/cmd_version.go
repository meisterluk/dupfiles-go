package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
)

// VersionCommand defines the CLI command parameters
type VersionCommand struct {
	ConfigOutput bool
	JSONOutput   bool
	Help         bool
}

// cliVersionCommand defines the CLI arguments as kingpin requires them
type cliVersionCommand struct {
	cmd          *kingpin.CmdClause
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

func newCLIVersionCommand(app *kingpin.Application) *cliVersionCommand {
	c := new(cliVersionCommand)
	c.cmd = app.Command("version", "Print implementation version, license and author. Exit code is always 0.")

	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *cliVersionCommand) Validate() (*VersionCommand, error) {
	// validity checks (check conditions not covered by kingpin)
	// (nothing.)

	// migrate cliVersionCommand to versionCommand
	cmd := new(VersionCommand)
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// default values
	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}
