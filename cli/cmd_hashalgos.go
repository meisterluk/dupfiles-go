package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
)

// HashAlgosCommand defines the CLI command parameters
type HashAlgosCommand struct {
	CheckSupport string `json:"check-support"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

// cliHashAlgosCommand defines the CLI arguments as kingpin requires them
type cliHashAlgosCommand struct {
	cmd          *kingpin.CmdClause
	CheckSupport *string
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

func newCLIHashAlgosCommand(app *kingpin.Application) *cliHashAlgosCommand {
	c := new(cliHashAlgosCommand)
	c.cmd = app.Command("hashalgos", "List supported hash algorithms.")

	c.CheckSupport = c.cmd.Flag("check-support", "exit code 1 indicates that the given hashalgo is unsupported").String()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *cliHashAlgosCommand) Validate() (*HashAlgosCommand, error) {
	// validity checks (check conditions not covered by kingpin)
	// (nothing.)

	// migrate CLIHashAlgosCommand to HashAlgosCommand
	cmd := new(HashAlgosCommand)
	cmd.CheckSupport = *c.CheckSupport
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput
	cmd.Help = false

	// default values
	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}
