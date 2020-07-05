package main

import (
	"encoding/json"
	"fmt"

	"github.com/meisterluk/dupfiles-go/internals"
	v1 "github.com/meisterluk/dupfiles-go/v1"
	"github.com/spf13/cobra"
)

// VersionCommand defines the CLI command parameters
type VersionCommand struct {
	CheckSupport string `json:"check-hashalgo-support"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

// VersionJSONResult is a struct used to serialize JSON output
type VersionJSONResult struct {
	Version     string              `json:"version"`
	Spec        string              `json:"api-version"`
	ReleaseDate string              `json:"release-date"`
	License     string              `json:"license"`
	Author      string              `json:"author"`
	HashAlgos   []HashAlgorithmData `json:"hash-algorithms"`
	Feedback    string              `json:"feedback"`
	Bugs        string              `json:"bugs"`
}

// HashAlgorithmData contains the metadata of a hash algorithm
type HashAlgorithmData struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Default bool   `json:"default"`
}

var versionCommand *VersionCommand
var argCheckSupport string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "returns metadata about this implementation",
	Long: `Returns the implementation's

• version
• specification implemented
• license name
• author name
• list of supported hash algorithms
• URL to report bugs
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global VersionCommand instance
	// called versionCommand and fill it with admissible parameters to run the version command.
	// It EITHER succeeds, fill versionCommand appropriately and returns nil.
	// OR returns an error instance and versionCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// create global VersionCommand instance
		versionCommand = new(VersionCommand)
		versionCommand.CheckSupport = argCheckSupport
		versionCommand.ConfigOutput = argConfigOutput
		versionCommand.JSONOutput = argJSONOutput
		versionCommand.Help = false

		// handle environment variables
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			versionCommand.JSONOutput = envJSON
			// NOTE ↓ ugly hack, to make Execute() return the appropriate value
			argJSONOutput = envJSON
		}

		return nil
	},
	// Run the version subcommand with versionCommand.
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, versionCommand}
		exitCode, cmdError = versionCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.PersistentFlags().StringVar(&argCheckSupport, "check-support", "", "exit code 100 indicates that the given hashalgo is unsupported")
}

const humanReadableRepresentation = `version:           %s
API implemented:   %s
release date:      %s
license:           %s
author:            %s
send feedback to:  %s
report bugs to:    %s

hash algorithms:
(* denotes default algorithm)
`

// Run executes the CLI command version on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *VersionCommand) Run(w, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	// fill VersionJSONResult with data
	data := VersionJSONResult{}
	data.Version = fmt.Sprintf("%d.%d.%d", v1.VERSION_MAJOR, v1.VERSION_MINOR, v1.VERSION_PATCH)
	data.Spec = fmt.Sprintf("%d.%d.%d", v1.SPEC_MAJOR, v1.SPEC_MINOR, v1.SPEC_PATCH)
	data.ReleaseDate = v1.RELEASE_DATE
	data.License = v1.LICENSE
	data.Author = `meisterluk`
	data.Feedback = `admin@lukas-prokop.at`
	data.Bugs = `https://github.com/meisterluk/dupfiles-go/issues/`

	data.HashAlgos = make([]HashAlgorithmData, 0, 8)
	for _, name := range (internals.HashAlgos{}.Names()) {
		status := "supported"
		if name == `crc64` || name == `fnv-1a-32` || name == `fnv-1a-128` || name == `sha-256` || name == `sha-512` || name == `sha-3-512` {
			status = "required"
		}
		data.HashAlgos = append(data.HashAlgos, HashAlgorithmData{
			Name:    name,
			Status:  status,
			Default: name == internals.HashAlgos{}.Default().Instance().Name(),
		})
	}

	// compute support check
	checkSupportFailed := false
	if c.CheckSupport != "" {
		for _, h := range (internals.HashAlgos{}.Names()) {
			if h == c.CheckSupport {
				checkSupportFailed = true
			}
		}
	}

	// compute output
	if c.JSONOutput {
		jsonRepr, err := json.MarshalIndent(&data, "", "  ")
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))
	} else {
		w.Printf(humanReadableRepresentation, data.Version, data.Spec, data.ReleaseDate, data.License, data.Author, data.Feedback, data.Bugs)
		for _, ha := range data.HashAlgos {
			isDefault := ""
			if ha.Default {
				isDefault = " *"
			}
			w.Printfln("\t%s%s  %s", ha.Name, isDefault, ha.Status)
		}
	}

	if c.CheckSupport != "" && checkSupportFailed {
		return 100, nil
	}

	return 0, nil
}
