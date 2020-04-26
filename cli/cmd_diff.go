package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

// DiffJSONObject represents one difference match of the diff command
type DiffJSONObject struct {
	Basename string   `json:"basename"`
	Digest   string   `json:"digest"`
	OccursIn []string `json:"occurs-in"`
}

// DiffJSONResult is a struct used to serialize JSON output
type DiffJSONResult struct {
	Children []DiffJSONObject `json:"children"`
}

// TargetPair contains the basenode and its associated report file.
// Pairs of these constitute the arguments you need to provide for subcommand diff.
type TargetPair struct {
	BaseNode string
	Report   string
}

// TargetPairs just implements the wrapper required by kingpin. See:
// https://github.com/alecthomas/kingpin/blob/b6657d9477a694/README.md#consuming-all-remaining-arguments
type TargetPairs []TargetPair

// Set adds the trailing argument provided as `value` to the set of parsed arguments
func (t *TargetPairs) Set(value string) error {
	if value == "" {
		return fmt.Errorf("'%s' is not a valid base node or report file", value)
	}
	// append to existing
	if len(*t) == 0 || ([]TargetPair)(*t)[len(*t)-1].Report != "" {
		*t = append(*t, TargetPair{BaseNode: value})
		return nil
	}
	// create new entry
	([]TargetPair)(*t)[len(*t)-1].Report = value
	return nil
}

// String returns a debug representation for the arguments
func (t *TargetPairs) String() string {
	out := "TargetPairs{"
	for _, target := range *t {
		out += fmt.Sprintf(`%s: %s, `, target.BaseNode, target.Report)
	}
	return out + "}"
}

// IsCumulative defines that TargetPairs consumes trailing arguments
func (t *TargetPairs) IsCumulative() bool {
	return true
}

// ParseTargets creates a new type for kingpin consuming trailing arguments
func ParseTargets(s *kingpin.ArgClause) *[]TargetPair {
	target := new([]TargetPair)
	s.SetValue((*TargetPairs)(target))
	return target
}

// CLIDiffCommand defines the CLI arguments as kingpin requires them
type CLIDiffCommand struct {
	cmd          *kingpin.CmdClause
	Targets      *[]TargetPair
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

// NewCLIDiffCommand defines the flags/arguments the CLI parser is supposed to understand
func NewCLIDiffCommand(app *kingpin.Application) *CLIDiffCommand {
	c := new(CLIDiffCommand)
	c.cmd = app.Command("diff", "Show difference between node children in two or more report files.")

	c.Targets = ParseTargets(c.cmd.Arg("targets", "two or more [{base node} {report}] pairs to consider"))
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

// Validate renders all arguments into a DiffCommand or throws an error.
// DiffCommand provides *all* arguments to run a 'diff' command.
func (c *CLIDiffCommand) Validate() (*DiffCommand, error) {
	// migrate CLIDiffCommand to DiffCommand
	cmd := new(DiffCommand)
	cmd.Targets = make([]TargetPair, 0, 8)
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// validate targets
	for _, target := range *c.Targets {
		if target.Report == "" {
			return cmd, fmt.Errorf(`base node '%s' needs the report file path it occurs in`, target.BaseNode)
		}
		for len(target.BaseNode) > 0 && target.BaseNode[len(target.BaseNode)-1] == filepath.Separator {
			target.BaseNode = target.BaseNode[:len(target.BaseNode)-1]
		}
		cmd.Targets = append(cmd.Targets, target)
	}
	if len(cmd.Targets) < 2 {
		return cmd, fmt.Errorf(`At least two [{base node} {report}] pairs are required for comparison, found %d`, len(cmd.Targets))
	}

	// handle environment variables
	envJSON, errJSON := EnvToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}

// DiffCommand defines the CLI command parameters
type DiffCommand struct {
	Targets      []TargetPair
	ConfigOutput bool
	JSONOutput   bool
	Help         bool
}

// Run executes the CLI command diff on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *DiffCommand) Run(w Output, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	type Identifier struct {
		Digest   string
		BaseName string
	}
	type match []bool
	type matches map[Identifier]match

	// use the first set to determine the entire set
	diffMatches := make(matches)
	anyFound := make([]bool, len(c.Targets))
	for t, match := range c.Targets {
		rep, err := internals.NewReportReader(match.Report)
		if err != nil {
			return 1, err
		}
		log.Printfln("# %s ⇒ %s", match.Report, match.BaseNode)
		for {
			tail, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				rep.Close()
				return 9, fmt.Errorf(`failure reading report file '%s' tailline: %s`, match.Report, err)
			}

			// TODO this assumes that paths are canonical and do not end with a folder separator
			if tail.Path == match.BaseNode && (tail.NodeType == 'D' || tail.NodeType == 'L') {
				anyFound[t] = true
			}
			if !strings.HasPrefix(tail.Path, match.BaseNode) || internals.DetermineDepth(tail.Path, rep.Head.Separator)-1 != internals.DetermineDepth(match.BaseNode, rep.Head.Separator) {
				continue
			}

			given := Identifier{Digest: internals.Hash(tail.HashValue).Digest(), BaseName: internals.Base(tail.Path, rep.Head.Separator)}
			value, ok := diffMatches[given]
			if ok {
				value[t] = true
			} else {
				diffMatches[given] = make([]bool, len(c.Targets))
				diffMatches[given][t] = true
			}
		}
		rep.Close()
	}

	if c.JSONOutput {
		data := DiffJSONResult{Children: make([]DiffJSONObject, 0, len(diffMatches))}
		for id, diffMatch := range diffMatches {
			occurences := make([]string, 0, len(c.Targets))
			for i, matches := range diffMatch {
				if matches {
					occurences = append(occurences, c.Targets[i].Report)
				}
			}
			data.Children = append(data.Children, DiffJSONObject{
				Basename: id.BaseName,
				Digest:   hex.EncodeToString([]byte(id.Digest)),
				OccursIn: occurences,
			})
		}

		jsonRepr, err := json.Marshal(&data)
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))

	} else {
		for i, anyMatch := range anyFound {
			if !anyMatch {
				log.Printf("# not found: '%s' in '%s'\n", c.Targets[i].Report, c.Targets[i].BaseNode)
			}
		}

		w.Println("")
		w.Println("# '+' means found, '-' means missing")

		for id, diffMatch := range diffMatches {
			for _, matched := range diffMatch {
				if matched {
					w.Printf("+")
				} else {
					w.Printf("-")
				}
			}
			w.Printfln("\t%s\t%s", hex.EncodeToString([]byte(id.Digest)), id.BaseName)
		}
	}

	return 0, nil
}
