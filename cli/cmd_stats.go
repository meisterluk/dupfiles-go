package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/meisterluk/dupfiles-go/internals"
	"github.com/spf13/cobra"
)

// StatsCommand defines the CLI command parameters
type StatsCommand struct {
	Report       string `json:"report"`
	Long         bool   `json:"long"`
	ConfigOutput bool   `json:"config"`
	JSONOutput   bool   `json:"json"`
	Help         bool   `json:"help"`
}

// BriefReportStatistics contains statistics collected from
// a report file and only requires single-pass parsing and
// constant memory to evaluate those statistics
type BriefReportStatistics struct {
	Head struct {
		Version       [3]uint16 `json:"version"`
		Timestamp     time.Time `json:"timestamp"`
		HashAlgorithm string    `json:"hash-algorithm"`
		ThreeMode     bool      `json:"three-mode"`
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

// SizeEntry represents a FilesOfMaxSize entry
type SizeEntry struct {
	Path string `json:"path"`
	Size uint64 `json:"size"`
}

// StatsJSONResult is a struct used to serialize JSON output
type StatsJSONResult struct {
	Brief BriefReportStatistics `json:"brief"`
	Long  LongReportStatistics  `json:"long"`
}

var statsCommand *StatsCommand
var argReport string
var argLong bool

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get statistics about a filesystem tree based on a report file",
	Long: `A report file encodes a filesystem state. This subcommand determines some statistics about the given filesystem.
For example: 

	dupfiles stats example_report.fsr
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global StatsCommand instance
	// called statsCommand and fill it with admissible parameters to run the stats command.
	// It EITHER succeeds, fill statsCommand appropriately and returns nil.
	// OR returns an error instance and statsCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// consider report as positional argument
		if len(args) > 1 {
			exitCode = 7
			return fmt.Errorf(`taking only one positional argument "report file"`)
		}
		if argReport == "" && len(args) == 0 {
			exitCode = 7
			return fmt.Errorf(`requires report file as positional argument`)
		} else if argReport != "" && len(args) == 0 {
			// ignore, argReport is properly set
		} else if argReport == "" && len(args) > 0 {
			argReport = args[0]
		} else if argReport != "" && len(args) > 0 {
			exitCode = 7
			return fmt.Errorf(`two report files supplied: "%s" and "%s"; expected only one`, argReport, args[0])
		}

		// create global StatsCommand instance
		statsCommand = new(StatsCommand)
		statsCommand.Report = argReport
		statsCommand.ConfigOutput = argConfigOutput
		statsCommand.JSONOutput = argJSONOutput
		statsCommand.Help = false

		// validity checks
		if statsCommand.Report == "" {
			exitCode = 8
			return fmt.Errorf("Report file argument must not be empty")
		}

		// handle environment variables
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			statsCommand.JSONOutput = envJSON
			// NOTE ↓ ugly hack, to make Execute() return the appropriate value
			argJSONOutput = envJSON
		}
		envLong, errLong := EnvToBool("DUPFILES_LONG")
		if errLong == nil {
			statsCommand.Long = envLong
		}

		return nil
	},
	// Run the stats subcommand with statsCommand
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, statsCommand}
		exitCode, cmdError = statsCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.PersistentFlags().StringVar(&argReport, `report`, "", `report to consider`)
	statsCmd.MarkFlagRequired("report")
	statsCmd.PersistentFlags().BoolVar(&argLong, `long`, false, `compute more features, but takes longer`)
}

// Run executes the CLI command stats on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *StatsCommand) Run(w, log Output) (int, error) {
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
	briefStats.Head.ThreeMode = rep.Head.ThreeMode
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
