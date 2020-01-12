package main

import (
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

// FindCommand defines the CLI command parameters
type FindCommand struct {
	Reports          []string `json:"reports"`
	Overwrite        bool     `json:"overwrite"`
	Output           string   `json:"output"`
	ResultByExitcode bool     `json:"result-by-exitcode"`
	ConfigOutput     bool     `json:"config"`
	JSONOutput       bool     `json:"json"`
	Help             bool     `json:"help"`
}

// cliFindCommand defined the CLI arguments as kingpin requires them
type cliFindCommand struct {
	cmd              *kingpin.CmdClause
	Reports          *[]string
	Overwrite        *bool
	Output           *string
	ResultByExitcode *bool
	ConfigOutput     *bool
	JSONOutput       *bool
}

func newCLIFindCommand(app *kingpin.Application) *cliFindCommand {
	c := new(cliFindCommand)
	c.cmd = app.Command("find", "Finds differences in report files.")

	c.Reports = c.cmd.Arg("reports", "reports to consider").Required().Strings()
	c.Overwrite = c.cmd.Flag("overwrite", "if filepath already exists, overwrite it without asking").Bool()
	c.Output = c.cmd.Flag("output", "write duplication results to file, not to stdout").Short('o').Default(envOr("DUPFILES_OUTPUT", "report.dup")).String()
	c.ResultByExitcode = c.cmd.Flag("result-by-exitcode", "use exit code 42 on success and if at least one duplicate was found").Bool()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *cliFindCommand) Validate() (*FindCommand, error) {
	// validity checks (check conditions not covered by kingpin)
	if len(*c.Reports) == 0 {
		return nil, fmt.Errorf("At least one report is required")
	}

	// migrate CLIFindCommand to FindCommand
	cmd := new(FindCommand)
	cmd.Reports = make([]string, len(*c.Reports))
	copy(cmd.Reports, *c.Reports)
	cmd.Overwrite = *c.Overwrite
	cmd.Output = *c.Output
	cmd.ResultByExitcode = *c.ResultByExitcode
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// default values
	envOverwrite, errOverwrite := envToBool("DUPFILES_OVERWRITE")
	if errOverwrite == nil {
		cmd.Overwrite = envOverwrite
	}
	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}
