package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

// targetPair contains the basenode and its associated report file.
// Pairs of these constitute the arguments you need to provide for subcommand diff.
type targetPair struct {
	BaseNode string
	Report   string
}

type targetPairs []targetPair

func (t *targetPairs) Set(value string) error {
	if value == "" {
		return fmt.Errorf("'%s' is not a valid base node or report file", value)
	}
	// append to existing
	if len(*t) == 0 || ([]targetPair)(*t)[len(*t)-1].Report != "" {
		*t = append(*t, targetPair{BaseNode: value})
		return nil
	}
	// create new entry
	([]targetPair)(*t)[len(*t)-1].Report = value
	return nil
}

func (t *targetPairs) String() string {
	out := "targetPairs{"
	for _, target := range *t {
		out += fmt.Sprintf(`%s: %s, `, target.BaseNode, target.Report)
	}
	return out + "}"
}

func (t *targetPairs) IsCumulative() bool {
	return true
}

func parseTargets(s *kingpin.ArgClause) *[]targetPair {
	target := new([]targetPair)
	s.SetValue((*targetPairs)(target))
	return target
}

// cliDiffCommand defines the CLI arguments as kingpin requires them
type cliDiffCommand struct {
	cmd          *kingpin.CmdClause
	Targets      *[]targetPair
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

func newCLIDiffCommand(app *kingpin.Application) *cliDiffCommand {
	c := new(cliDiffCommand)
	c.cmd = app.Command("diff", "Show difference between node children in two or more report files.")

	c.Targets = parseTargets(c.cmd.Arg("targets", "two or more [{base node} {report}] pairs to consider"))
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

func (c *cliDiffCommand) Validate() (*DiffCommand, error) {
	// migrate cliDiffCommand to DiffCommand
	cmd := new(DiffCommand)
	cmd.Targets = make([]targetPair, 0, 8)
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
	envJSON, errJSON := envToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}

	return cmd, nil
}

// DiffCommand defines the CLI command parameters
type DiffCommand struct {
	Targets      []targetPair
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

	// use the first set to determine the set
	diffMatches := make(matches)
	anyFound := make([]bool, len(c.Targets))
	for t, match := range c.Targets {
		rep, err := internals.NewReportReader(match.Report)
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(os.Stderr, "# %s ⇒ %s\n", match.Report, match.BaseNode)
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
			if filepath.Dir(tail.Path) != match.BaseNode {
				continue
			}

			given := Identifier{Digest: string(tail.HashValue), BaseName: filepath.Base(tail.Path)}
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
		type jsonObject struct {
			Basename string   `json:"basename"`
			Digest   string   `json:"digest"`
			OccursIn []string `json:"occurs-in"`
		}
		type jsonResult struct {
			Children []jsonObject `json:"children"`
		}

		data := jsonResult{Children: make([]jsonObject, 0, len(diffMatches))}
		for id, diffMatch := range diffMatches {
			occurences := make([]string, 0, len(c.Targets))
			for i, matches := range diffMatch {
				if matches {
					occurences = append(occurences, c.Targets[i].Report)
				}
			}
			data.Children = append(data.Children, jsonObject{
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
