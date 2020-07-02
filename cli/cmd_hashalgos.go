package main

import (
	"encoding/json"
	"fmt"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

// HashAlgosJSONResult is a struct used to serialize JSON output
type HashAlgosJSONResult struct {
	Check struct {
		Query     string `json:"query"`
		Supported bool   `json:"is-supported"`
	} `json:"support-check,omitempty"`
	Algos struct {
		Supported []string `json:"supported"`
		Default   string   `json:"default"`
	} `json:"algorithms"`
}

// CLIHashAlgosCommand defines the CLI arguments as kingpin requires them
type CLIHashAlgosCommand struct {
	cmd          *kingpin.CmdClause
	CheckSupport *string
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

// NewCLIHashAlgosCommand defines the flags/arguments the CLI parser is supposed to understand
func NewCLIHashAlgosCommand(app *kingpin.Application) *CLIHashAlgosCommand {
	c := new(CLIHashAlgosCommand)
	c.cmd = app.Command("hashalgos", "List supported hash algorithms.")

	c.CheckSupport = c.cmd.Flag("check-support", "exit code 1 indicates that the given hashalgo is unsupported").String()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

// Validate renders all arguments into a HashAlgosCommand or throws an error.
// HashAlgosCommand provides *all* arguments to run a 'hashalgos' command.
func (c *CLIHashAlgosCommand) Validate() (*HashAlgosCommand, error) {
	// migrate CLIHashAlgosCommand to HashAlgosCommand
	cmd := new(HashAlgosCommand)
	cmd.CheckSupport = *c.CheckSupport
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput
	cmd.Help = false

	// handle environment variables
	envJSON, errJSON := EnvToBool("DUPFILES_JSON")
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

	data := HashAlgosJSONResult{}
	data.Algos.Supported = internals.HashAlgos{}.Names()
	data.Algos.Default = internals.HashAlgos{}.Default().Instance().Name()

	if c.CheckSupport != "" {
		data.Check.Query = c.CheckSupport
		for _, h := range (internals.HashAlgos{}.Names()) {
			if h == c.CheckSupport {
				data.Check.Supported = true
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

	if c.CheckSupport != "" && !data.Check.Supported {
		return 100, nil
	}
	return 0, nil
}
