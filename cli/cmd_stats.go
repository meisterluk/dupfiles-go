package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

// SizeEntry represents a Top10MaxSizeFiles entry
type SizeEntry struct {
	Path string `json:"path"`
	Size uint64 `json:"size"`
}

// BriefReportStatistics contains statistics collected from
// a report file and only requires single-pass parsing and
// constant memory to evaluate those statistics
type BriefReportStatistics struct {
	HeadVersion         [3]uint16     `json:"head-version"`
	HeadTimestamp       time.Time     `json:"head-timestamp"`
	HeadHashAlgorithm   string        `json:"head-hash-algorithm"`
	HeadBasenameMode    bool          `json:"head-basename-mode"`
	HeadNodeName        string        `json:"head-node-name"`
	HeadBasePath        string        `json:"head-base-path"`
	NumUNIXDeviceFile   uint32        `json:"count-unix-device"`
	NumDirectory        uint32        `json:"count-directory"`
	NumRegularFile      uint32        `json:"count-regular-file"`
	NumLink             uint32        `json:"count-link"`
	NumFIFOPipe         uint32        `json:"count-fifo-pipe"`
	NumUNIXDomainSocket uint32        `json:"count-unix-socket"`
	MaxDepth            uint16        `json:"fs-depth-max"`
	TotalSize           uint64        `json:"fs-size-total"`
	Top10MaxSizeFiles   [10]SizeEntry `json:"files-size-max-top10"`
}

// LongReportStatistics contains statistics collected from
// a report file and requires linear time and linear memory
// (or more) to evaluate those statistics
type LongReportStatistics struct {
	// {average, median, min, max} number of children in a folder?
}

// StatsJSONResult is a struct used to serialize JSON output
type StatsJSONResult struct {
	Brief BriefReportStatistics `json:"brief"`
	Long  LongReportStatistics  `json:"long"`
}

// CLIStatsCommand defines the CLI arguments as kingpin requires them
type CLIStatsCommand struct {
	cmd          *kingpin.CmdClause
	Report       *string
	Long         *bool
	ConfigOutput *bool
	JSONOutput   *bool
	Help         *bool
}

// NewCLIStatsCommand defines the flags/arguments the CLI parser is supposed to understand
func NewCLIStatsCommand(app *kingpin.Application) *CLIStatsCommand {
	c := new(CLIStatsCommand)
	c.cmd = app.Command("stats", "Prints some statistics about filesystem nodes based on a report.")

	c.Report = c.cmd.Arg("report", "report to consider").Required().String()
	c.Long = c.cmd.Flag("long", "compute more features, but takes longer").Bool()
	c.ConfigOutput = c.cmd.Flag("config", "only prints the configuration and terminates").Bool()
	c.JSONOutput = c.cmd.Flag("json", "return output as JSON, not as plain text").Bool()

	return c
}

// Validate renders all arguments into a StatsCommand or throws an error.
// StatsCommand provides *all* arguments to run a 'stats' command.
func (c *CLIStatsCommand) Validate() (*StatsCommand, error) {
	// validity checks (check conditions that are not covered by kingpin)
	if *c.Report == "" {
		return nil, fmt.Errorf("One report must be specified")
	}

	// migrate CLIStatsCommand to StatsCommand
	cmd := new(StatsCommand)
	cmd.Report = *c.Report
	cmd.ConfigOutput = *c.ConfigOutput
	cmd.JSONOutput = *c.JSONOutput

	// handle environment variables
	envJSON, errJSON := EnvToBool("DUPFILES_JSON")
	if errJSON == nil {
		cmd.JSONOutput = envJSON
	}
	envLong, errLong := EnvToBool("DUPFILES_LONG")
	if errLong == nil {
		cmd.Long = envLong
	}

	return cmd, nil
}

// StatsCommand defines the CLI command parameters
type StatsCommand struct {
	Report       string `json:"report"`
	Long         bool   `json:"long"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

// Run executes the CLI command stats on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *StatsCommand) Run(w Output, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	rep, err := internals.NewReportReader(c.Report)
	if err != nil {
		return 1, fmt.Errorf(`failure reading report file '%s': %s`, c.Report, err)
	}
	var briefStats BriefReportStatistics
	for {
		tail, err := rep.Iterate()
		if err == io.EOF {
			break
		}
		if err != nil {
			rep.Close()
			return 9, fmt.Errorf(`failure reading report file '%s' tailline: %s`, c.Report, err)
		}

		// consider node type
		switch tail.NodeType {
		case 'D':
			briefStats.NumDirectory++
		case 'C':
			briefStats.NumUNIXDeviceFile++
		case 'F':
			briefStats.NumRegularFile++
		case 'L':
			briefStats.NumLink++
		case 'P':
			briefStats.NumFIFOPipe++
		case 'S':
			briefStats.NumUNIXDomainSocket++
		default:
			return 9, fmt.Errorf(`unknown node type '%c'`, tail.NodeType)
		}

		// consider folder depth
		depth := internals.DetermineDepth(tail.Path)
		if depth > briefStats.MaxDepth {
			briefStats.MaxDepth = depth
		}

		// consider size
		briefStats.TotalSize += tail.FileSize
		oldTotalSize := briefStats.TotalSize
		if oldTotalSize > briefStats.TotalSize {
			return 11, fmt.Errorf(`total-size overflowed from %d to %d`, oldTotalSize, briefStats.TotalSize)
		}

		for i := 0; i < 10; i++ {
			if tail.NodeType == 'D' {
				continue
			}
			if briefStats.Top10MaxSizeFiles[i].Size > tail.FileSize {
				continue
			}
			tmp := briefStats.Top10MaxSizeFiles[i]
			briefStats.Top10MaxSizeFiles[i].Size = tail.FileSize
			briefStats.Top10MaxSizeFiles[i].Path = tail.Path
			for j := i + 1; j < 10; j++ {
				tmp2 := briefStats.Top10MaxSizeFiles[j]
				briefStats.Top10MaxSizeFiles[j] = tmp
				tmp = tmp2
			}
			break
		}
	}

	// report Head data
	briefStats.HeadVersion = rep.Head.Version
	briefStats.HeadTimestamp = rep.Head.Timestamp
	briefStats.HeadHashAlgorithm = rep.Head.HashAlgorithm
	briefStats.HeadBasenameMode = rep.Head.BasenameMode
	briefStats.HeadNodeName = rep.Head.NodeName
	briefStats.HeadBasePath = rep.Head.BasePath

	var longStats LongReportStatistics
	if c.Long {
		// which data will be evaluated here?
	}

	var out StatsJSONResult
	out.Brief = briefStats
	out.Long = longStats

	if c.JSONOutput {
		jsonRepr, err := json.Marshal(&out)
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))
	} else {
		jsonRepr, err := json.MarshalIndent(&out, "", "  ")
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))
	}

	rep.Close()

	return 0, nil
}
