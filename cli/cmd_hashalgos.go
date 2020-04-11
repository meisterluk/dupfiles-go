package main

import (
	"encoding/json"
	"fmt"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

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
	// migrate CLIHashAlgosCommand to HashAlgosCommand
	cmd := new(HashAlgosCommand)
	cmd.CheckSupport = *c.CheckSupport
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput
	cmd.Help = false

	// handle environment variables
	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}

// HashAlgosCommand defines the CLI command parameters
type HashAlgosCommand struct {
	CheckSupport string `json:"check-support"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

// Run executes the CLI command hashalgos on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *HashAlgosCommand) Run(w Output, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	type dataSet struct {
		CheckSucceeded bool     `json:"check-result"`
		SupHashAlgos   []string `json:"supported-hash-algorithms"`
	}

	data := dataSet{
		CheckSucceeded: false,
		SupHashAlgos:   internals.SupportedHashAlgorithms(),
	}

	if c.CheckSupport != "" {
		for _, h := range internals.SupportedHashAlgorithms() {
			if h == c.CheckSupport {
				data.CheckSucceeded = true
			}
		}
	}

	if c.JSONOutput {
		b, err := json.Marshal(&data)
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(b))
	} else {
		jsonRepr, err := json.MarshalIndent(&data, "", "  ")
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))
	}
}
