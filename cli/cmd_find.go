package main

import (
	"fmt"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
)

// FindCommand defines the CLI command parameters
type FindCommand struct {
	Reports          []string `json:"reports"`
	Strategy         []string `json:"strategy"`
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
	Strategy         *string
	Overwrite        *bool
	Output           *string
	ResultByExitcode *bool
	ConfigOutput     *bool
	JSONOutput       *bool
	Help             *bool
}

func newCLIFindCommand(app *kingpin.Application) *cliFindCommand {
	c := new(cliFindCommand)
	c.cmd = app.Command("find", "Finds differences in report files.")

	c.Reports = c.cmd.Arg("reports", "reports to consider").Required().Strings()
	c.Strategy = c.cmd.Flag("strategy", "comparison strategy (e.g. 'filesize,hash')").Short('s').Default(envOr("DUPFILES_STRATEGY", "filesize,hash")).String()
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
	strategyUnits := strings.Split(*c.Strategy, ",")
	strategy := make([]string, 0) // must be ordered set with whitelisted values
	for _, unit := range strategyUnits {
		found := false
		for _, s := range strategy {
			if unit == s {
				found = true
			}
			if unit != `filesize` && unit != `hash` {
				return nil, fmt.Errorf(`Strategy must be 'filesize' or 'hash', not '%s' in '%s'`, unit, *c.Strategy)
			}
		}
		if !found {
			strategy = append(strategy, unit)
		}
	}

	// migrate CLIFindCommand to FindCommand
	cmd := new(FindCommand)
	cmd.Reports = make([]string, 0)

	copy(cmd.Reports, *c.Reports)
	copy(cmd.Strategy, strategy)
	cmd.Overwrite = *c.Overwrite
	cmd.Output = *c.Output
	cmd.ResultByExitcode = *c.ResultByExitcode
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput
	cmd.Help = *c.Help

	// default values
	if envToBool("DUPFILES_OVERWRITE") {
		cmd.Overwrite = true
	}
	if envToBool("DUPFILES_JSON") {
		cmd.JSONOutput = true
	}

	return cmd, nil
}
