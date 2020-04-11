package main

import (
	"encoding/json"
	"fmt"

	v1 "github.com/meisterluk/dupfiles-go/v1"
	"gopkg.in/alecthomas/kingpin.v2"
)

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
	// migrate cliVersionCommand to versionCommand
	cmd := new(VersionCommand)
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// handle environment variables
	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}

// VersionCommand defines the CLI command parameters
type VersionCommand struct {
	ConfigOutput bool
	JSONOutput   bool
	Help         bool
}

// Run executes the CLI command version on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *VersionCommand) Run(w Output, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	versionString := fmt.Sprintf("%d.%d.%d", v1.VERSION_MAJOR, v1.VERSION_MINOR, v1.VERSION_PATCH)

	if c.JSONOutput {
		type jsonResult struct {
			Version string `json:"version"`
		}

		data := jsonResult{Version: versionString}
		b, err := json.Marshal(&data)
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(b))

	} else {
		w.Println(versionString)
	}

	return 0, nil
}
