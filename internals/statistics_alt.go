package internals

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Analysis contains computed data about the file system state
type Analysis struct {
	MaxDepth              uint64
	TotalByteSize         uint64
	TotalEntries          uint64
	CountFiles            uint64
	CountDirectory        uint64
	CountDeviceFile       uint64
	CountLink             uint64
	CountFIFOPipe         uint64
	CountUNIXDomainSocket uint64
}

func (a *Analysis) String() string {
	bytesPerFile := uint64(0)
	if a.CountFiles > 0 {
		bytesPerFile = a.TotalByteSize / a.CountFiles
	}
	return fmt.Sprintf(`maxdepth=%d bytes=%s #=%d ratio=%s/file  #files=%d #dirs=%d #devicefiles=%d #links=%d #pipes=%d #socket=%d`,
		a.MaxDepth, HumanReadableBytes(a.TotalByteSize), a.TotalEntries, HumanReadableBytes(bytesPerFile),
		a.CountFiles, a.CountDirectory, a.CountDeviceFile, a.CountLink, a.CountFIFOPipe,
		a.CountUNIXDomainSocket,
	)
}

// DoAnalysis runs the evaluation of the file system state.
// The analysis is parameterized by all the other arguments given here.
func DoAnalysis(data *Analysis, depth uint64, baseNode string, ignorePermErrors bool, excludeFilename []string, excludeFilenameRegexp []*regexp.Regexp, excludeTree []string) error {
	for _, tree := range excludeTree {
		if len(tree) > 0 && strings.HasSuffix(baseNode, tree) {
			return nil
		}
	}

	stat, err := os.Stat(baseNode)
	// is a file
	if err == nil && !stat.IsDir() {
		baseName := stat.Name()
		if Contains(excludeFilename, baseName) {
			return nil
		}
		for _, re := range excludeFilenameRegexp {
			if re.FindString(baseName) != "" {
				return nil
			}
		}

		data.TotalByteSize += uint64(stat.Size())
		data.TotalEntries++

		switch DetermineNodeType(stat) {
		case 'C':
			data.CountDeviceFile++
		case 'F':
			data.CountFiles++
		case 'L':
			data.CountLink++
		case 'P':
			data.CountFIFOPipe++
		case 'S':
			data.CountUNIXDomainSocket++
		case 'X':
			return fmt.Errorf(`Unknown node type for %s`, baseNode)
		}
		return nil

		// is a directory
	} else if err == nil {
		data.TotalEntries++
		data.CountDirectory++
		depth++

		if depth > data.MaxDepth {
			data.MaxDepth = depth
		}

		entries, err := os.ReadDir(baseNode)
		if err != nil {
			return err
		}

		for _, fileInfo := range entries {
			err = DoAnalysis(data, depth, filepath.Join(baseNode, fileInfo.Name()), ignorePermErrors, excludeFilename, excludeFilenameRegexp, excludeTree)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// some error like 'does not exist' occured
	return fmt.Errorf(`error for %s: %s`, baseNode, err.Error())
}

// Analyze provides the API call to perform an analysis.
func Analyze(baseNode string, ignorePermErrors bool, excludeFilename []string, excludeFilenameRegex []string, excludeTree []string) (Analysis, error) {
	var data Analysis

	excludeFilenameRegexp := make([]*regexp.Regexp, 0)
	for _, regex := range excludeFilenameRegex {
		re, err := regexp.CompilePOSIX(regex)
		if err != nil {
			return data, fmt.Errorf(`regex '%s' invalid: %s`, regex, err.Error())
		}
		excludeFilenameRegexp = append(excludeFilenameRegexp, re)
	}

	err := DoAnalysis(&data, 0, baseNode, ignorePermErrors, excludeFilename, excludeFilenameRegexp, excludeTree)
	if err != nil {
		return data, err
	}
	return data, nil
}
