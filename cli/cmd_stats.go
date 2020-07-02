package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/meisterluk/dupfiles-go/internals"
	"gopkg.in/alecthomas/kingpin.v2"
)

// SizeEntry represents a FilesOfMaxSize entry
type SizeEntry struct {
	Path string `json:"path"`
	Size uint64 `json:"size"`
}

// BriefReportStatistics contains statistics collected from
// a report file and only requires single-pass parsing and
// constant memory to evaluate those statistics
type BriefReportStatistics struct {
	Head struct {
		Version       [3]uint16 `json:"version"`
		Timestamp     time.Time `json:"timestamp"`
		HashAlgorithm string    `json:"hash-algorithm"`
		BasenameMode  bool      `json:"basename-mode"`
		Separator     string    `json:"separator"`
		NodeName      string    `json:"node-name"`
		BasePath      string    `json:"base-path"`
	} `json:"head"`
	TotalCount struct {
		NumUNIXDeviceFile   uint32 `json:"unix-device"`
		NumDirectory        uint32 `json:"directory"`
		NumRegularFile      uint32 `json:"regular-file"`
		NumLink             uint32 `json:"link"`
		NumFIFOPipe         uint32 `json:"fifo-pipe"`
		NumUNIXDomainSocket uint32 `json:"unix-socket"`
	} `json:"total-count"`
	MaxDepth        uint16        `json:"max-fsdepth"`
	AccumulatedSize uint64        `json:"sum-size"`
	FilesOfMaxSize  [10]SizeEntry `json:"max-size-files"`
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
		tail, _, err := rep.Iterate()
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
			briefStats.TotalCount.NumDirectory++
		case 'C':
			briefStats.TotalCount.NumUNIXDeviceFile++
		case 'F':
			briefStats.TotalCount.NumRegularFile++
		case 'L':
			briefStats.TotalCount.NumLink++
		case 'P':
			briefStats.TotalCount.NumFIFOPipe++
		case 'S':
			briefStats.TotalCount.NumUNIXDomainSocket++
		default:
			return 9, fmt.Errorf(`unknown node type '%c'`, tail.NodeType)
		}

		// consider folder depth
		depth := internals.DetermineDepth(tail.Path, rep.Head.Separator)
		if depth > briefStats.MaxDepth {
			briefStats.MaxDepth = depth
		}

		// consider size
		briefStats.AccumulatedSize += tail.FileSize
		oldAccumulatedSize := briefStats.AccumulatedSize
		if oldAccumulatedSize > briefStats.AccumulatedSize {
			return 11, fmt.Errorf(`total-size overflowed from %d to %d`, oldAccumulatedSize, briefStats.AccumulatedSize)
		}

		for i := 0; i < 10; i++ {
			if tail.NodeType == 'D' {
				continue
			}
			if briefStats.FilesOfMaxSize[i].Size > tail.FileSize {
				continue
			}
			tmp := briefStats.FilesOfMaxSize[i]
			briefStats.FilesOfMaxSize[i].Size = tail.FileSize
			briefStats.FilesOfMaxSize[i].Path = tail.Path
			for j := i + 1; j < 10; j++ {
				tmp2 := briefStats.FilesOfMaxSize[j]
				briefStats.FilesOfMaxSize[j] = tmp
				tmp = tmp2
			}
			break
		}
	}

	// report Head data
	briefStats.Head.Version = rep.Head.Version
	briefStats.Head.Timestamp = rep.Head.Timestamp
	briefStats.Head.HashAlgorithm = rep.Head.HashAlgorithm
	briefStats.Head.BasenameMode = rep.Head.BasenameMode
	briefStats.Head.Separator = string(rep.Head.Separator)
	briefStats.Head.NodeName = rep.Head.NodeName
	briefStats.Head.BasePath = rep.Head.BasePath

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
		// TODO plain text output
		jsonRepr, err := json.MarshalIndent(&out, "", "  ")
		if err != nil {
			return 6, fmt.Errorf(resultJSONErrMsg, err)
		}
		w.Println(string(jsonRepr))
	}

	rep.Close()

	return 0, nil
}
