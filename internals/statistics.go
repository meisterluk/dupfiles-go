package internals

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// Statistics collects data of the preevaluation
type Statistics struct {
	MaxSize      uint64
	MaxDepth     uint16
	CountFiles   uint32
	CountFolders uint32
	ErrorMessage error
}

func (s *Statistics) String() string {
	if s.ErrorMessage != nil {
		return fmt.Sprintf(`stats: an error occured - %s`, s.ErrorMessage.Error())
	}
	d := "dirs"
	if s.CountFolders == 1 {
		d = "dir"
	}
	f := "files"
	if s.CountFiles == 1 {
		f = "file"
	}
	return fmt.Sprintf(`stats: %d %s %d %s %s maxsize %d maxdepth`, s.CountFolders, d, s.CountFiles, f, humanReadableBytes(s.MaxSize), s.MaxDepth)
}

func (s *Statistics) Error() string {
	if s.ErrorMessage != nil {
		return s.ErrorMessage.Error()
	}
	return ""
}

// GenerateStatistics determines Statistics for the gives base node.
// It serves as a pre-evaluation of the file system.
// Because it does not read the content of files, it is expected to be much faster than the hashing walk.
// The result is written to the provided channel.
func GenerateStatistics(baseNode string, ignorePermErrors bool, excludeBasename, excludeBasenameRegex, excludeTree []string) Statistics {
	var stats Statistics

	regexes := make([]*regexp.Regexp, len(excludeBasenameRegex))
	for _, r := range excludeBasenameRegex {
		regexes = append(regexes, regexp.MustCompilePOSIX(r))
	}

	stats.ErrorMessage = filepath.Walk(baseNode, func(path string, info os.FileInfo, err error) error {
		// if error occured, handle it
		if err != nil {
			if isPermissionError(err) && ignorePermErrors {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}

		// INVARIANT err == nil

		// run exclusion checks
		reject := false

		for i, t := range excludeTree {
			if path == t {
				log.Printf(`'%s' matches tree '%s'`, path, excludeTree[i])
				reject = true
			}
		}

		if !reject {
			for i, s := range excludeBasename {
				if info.Name() == s {
					log.Printf(`'%s' matches basename '%s'`, path, excludeBasename[i])
					reject = true
				}
			}
		}

		if !reject {
			for i, r := range regexes {
				if r.MatchString(info.Name()) {
					log.Printf(`'%s' matches regex '%s'`, info.Name(), excludeBasenameRegex[i])
					reject = true
				}
			}
		}

		if reject {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// handle size
		_size := info.Size()
		if _size < 0 {
			_size = 0
		}
		size := uint64(_size)
		if stats.MaxSize < size {
			stats.MaxSize = size
		}

		// increase counters
		if info != nil {
			if info.IsDir() {
				stats.CountFolders++
			} else {
				stats.CountFiles++
			}
		}

		// handle depth
		depth := DetermineDepth(path)
		if depth > stats.MaxDepth {
			stats.MaxDepth = depth
		}

		return nil
	})

	return stats
}
