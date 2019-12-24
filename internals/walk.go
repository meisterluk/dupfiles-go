package internals

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func classify(stat os.FileInfo) rune {
	mode := stat.Mode()
	switch {
	case mode&os.ModeDevice != 0:
		return 'C'
	case mode.IsDir():
		return 'D'
	case mode.IsRegular():
		return 'F'
	case mode&os.ModeSymlink != 0:
		return 'L'
	case mode&os.ModeNamedPipe != 0:
		return 'P'
	case mode&os.ModeSocket != 0:
		return 'S'
	}
	return 'X'
}

func walkBFS(baseNode string, ignorePermErrors bool, excludeFilename []string, excludeFilenameRegexp []*regexp.Regexp, excludeTree []string, out chan<- string) error {
	if contains(excludeTree, baseNode) {
		return nil
	}

	entries, err := ioutil.ReadDir(baseNode)
	if err != nil {
		if isPermissionError(err) && ignorePermErrors {
			return nil
		}
		return err
	}

	dirs := make([]string, 0, 8)

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(baseNode, entry.Name()))
		} else {
			if contains(excludeFilename, entry.Name()) {
				continue
			}
			for _, re := range excludeFilenameRegexp {
				if re.FindString(entry.Name()) != "" {
					continue
				}
			}
			out <- filepath.Join(baseNode, entry.Name())
		}
	}

	for _, dir := range dirs {
		err := walkBFS(dir, ignorePermErrors, excludeFilename, excludeFilenameRegexp, excludeTree, out)
		if err != nil {
			return err
		}
		if contains(excludeTree, dir) {
			continue
		}
		out <- dir
	}

	return nil
}

func walkDFS(baseNode string, ignorePermErrors bool, excludeFilename []string, excludeFilenameRegexp []*regexp.Regexp, excludeTree []string, out chan<- string) error {
	if contains(excludeTree, baseNode) {
		return nil
	}

	entries, err := ioutil.ReadDir(baseNode)
	if err != nil {
		if isPermissionError(err) && ignorePermErrors {
			return nil
		} else {
			return err
		}
	}

	nondirs := make([]string, 0, 8)

	for _, entry := range entries {
		joined := filepath.Join(baseNode, entry.Name())
		if entry.IsDir() {
			err := walkDFS(joined, ignorePermErrors, excludeFilename, excludeFilenameRegexp, excludeTree, out)
			if err != nil {
				return err
			}
			if contains(excludeTree, joined) {
				continue
			}
			out <- joined
		} else {
			if contains(excludeFilename, entry.Name()) {
				continue
			}
			for _, re := range excludeFilenameRegexp {
				if re.FindString(entry.Name()) != "" {
					continue
				}
			}
			nondirs = append(nondirs, joined)
		}
	}

	for _, nondir := range nondirs {
		out <- nondir
	}

	return nil
}

func Walk(baseNode string, bfs bool, ignorePermErrors bool, excludeFilename []string, excludeFilenameRegex []string, excludeTree []string, out chan string) error {
	// analyze folder structure before traversal
	analysis, err := analyze(baseNode, ignorePermErrors, excludeFilename, excludeFilenameRegex, excludeTree)
	if err != nil {
		return err
	}

	fmt.Println(analysis.String())

	// prepare parameters
	excludeFilenameRegexp := make([]*regexp.Regexp, 0, 8)
	for _, regex := range excludeFilenameRegex {
		re, err := regexp.CompilePOSIX(regex)
		if err != nil {
			return fmt.Errorf(`regex '%s' invalid: %s`, regex, err.Error())
		}
		excludeFilenameRegexp = append(excludeFilenameRegexp, re)
	}

	// baseNode points to a single file?
	fileInfo, err := os.Stat(baseNode)
	if err != nil {
		if ignorePermErrors && isPermissionError(err) {
			return nil
		}
		return err
	} else if !fileInfo.IsDir() {
		if contains(excludeFilename, baseNode) {
			return nil
		}
		if contains(excludeTree, baseNode) {
			return nil
		}
		for _, re := range excludeFilenameRegexp {
			if re.FindString(baseNode) != "" {
				return nil
			}
		}
		out <- baseNode
		return nil
	}

	// recursive traversal
	if bfs {
		err = walkBFS(baseNode, ignorePermErrors, excludeFilename, excludeFilenameRegexp, excludeTree, out)
	} else {
		err = walkDFS(baseNode, ignorePermErrors, excludeFilename, excludeFilenameRegexp, excludeTree, out)
	}
	if err == nil {
		out <- baseNode
	}
	close(out)
	return err
}
