package internals

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
		a.MaxDepth, humanReadableBytes(a.TotalByteSize), a.TotalEntries, humanReadableBytes(bytesPerFile),
		a.CountFiles, a.CountDirectory, a.CountDeviceFile, a.CountLink, a.CountFIFOPipe,
		a.CountUNIXDomainSocket,
	)
}

func contains(set []string, item string) bool {
	for _, element := range set {
		if item == element {
			return true
		}
	}
	return false
}

func doAnalysis(data *Analysis, depth uint64, baseNode string, ignorePermErrors bool, excludeFilename []string, excludeFilenameRegexp []*regexp.Regexp, excludeTree []string) error {
	for _, tree := range excludeTree {
		if len(tree) > 0 && strings.HasSuffix(baseNode, tree) {
			return nil
		}
	}

	stat, err := os.Stat(baseNode)
	// is a file
	if err == nil && !stat.IsDir() {
		baseName := stat.Name()
		if contains(excludeFilename, baseName) {
			return nil
		}
		for _, re := range excludeFilenameRegexp {
			if re.FindString(baseName) != "" {
				return nil
			}
		}

		data.TotalByteSize += uint64(stat.Size())
		data.TotalEntries += 1

		switch classify(stat) {
		case 'C':
			data.CountDeviceFile += 1
		case 'F':
			data.CountFiles += 1
		case 'L':
			data.CountLink += 1
		case 'P':
			data.CountFIFOPipe += 1
		case 'S':
			data.CountUNIXDomainSocket += 1
		case 'X':
			return fmt.Errorf(`Unknown node type for %s`, baseNode)
		}
		return nil

		// is a directory
	} else if err == nil {
		data.TotalEntries += 1
		data.CountDirectory += 1
		depth += 1

		if depth > data.MaxDepth {
			data.MaxDepth = depth
		}

		entries, err := ioutil.ReadDir(baseNode)
		if err != nil {
			return err
		}

		for _, fileInfo := range entries {
			err = doAnalysis(data, depth, filepath.Join(baseNode, fileInfo.Name()), ignorePermErrors, excludeFilename, excludeFilenameRegexp, excludeTree)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// some error like 'does not exist' occured
	return fmt.Errorf(`error for %s: %s`, baseNode, err.Error())
}

func analyze(baseNode string, ignorePermErrors bool, excludeFilename []string, excludeFilenameRegex []string, excludeTree []string) (Analysis, error) {
	var data Analysis

	excludeFilenameRegexp := make([]*regexp.Regexp, 0)
	for _, regex := range excludeFilenameRegex {
		re, err := regexp.CompilePOSIX(regex)
		if err != nil {
			return data, fmt.Errorf(`regex '%s' invalid: %s`, regex, err.Error())
		}
		excludeFilenameRegexp = append(excludeFilenameRegexp, re)
	}

	err := doAnalysis(&data, 0, baseNode, ignorePermErrors, excludeFilename, excludeFilenameRegexp, excludeTree)
	if err != nil {
		return data, err
	}
	return data, nil
}
