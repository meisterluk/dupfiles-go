package internals

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func NewReportReader(filepath string) (*Report, error) {
	reportFile := new(Report)
	reportFile.FilePath = filepath
	if filepath == "-" {
		reportFile.File = os.Stdin
	} else {
		fd, err := os.Open(filepath)
		if err != nil {
			return nil, err
		}
		reportFile.File = fd
	}
	return reportFile, nil
}

// Iterate reads and parses the next tail line in the file
func (r *Report) Iterate() (ReportTailLine, error) {
	tail := ReportTailLine{}
	tailLineRead := false

	for {
		// read one line from the file
		eofMet := false
		var cache [1]byte
		var buffer [512]byte
		bufferIndex := 0
		for {
			_, err := r.File.Read(cache[:])
			if err != io.EOF {
				if err != nil {
					return tail, err
				}
				if bufferIndex > 0 || (cache[0] != '\n' && cache[0] != '\r') {
					buffer[bufferIndex] = cache[0]
					bufferIndex++
					if bufferIndex == 512 {
						return tail, fmt.Errorf(`line too long, please report this issue to the developers`)
					}
				}
			} else {
				eofMet = true
				break
			}
			if bufferIndex > 0 && cache[0] == '\n' {
				break
			}
		}

		if bufferIndex == 0 && eofMet {
			return tail, io.EOF
		}

		if buffer[0] == '#' && r.Head.HashAlgorithm == "" {
			// parse head line
			regex, err := regexp.CompilePOSIX(`# +([0-9.]+(\.[0-9.]+){0,2}) +([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}) +([-_a-zA-Z0-9]+) (B|E) +([-_a-zA-Z0-9]+) +([^\r\n]+)`)
			if err != nil {
				return tail, err
			}

			groups := regex.FindSubmatch(buffer[0:bufferIndex])
			versionNumber, err := parseVersionNumber(string(groups[1]))
			if err != nil {
				return tail, err
			}

			timestamp, err := parseTimestamp(string(groups[3]))
			if err != nil {
				return tail, err
			}

			hashAlgorithm := strings.ToLower(string(groups[4]))
			if !isValidHashAlgo(hashAlgorithm) {
				return tail, fmt.Errorf(`Unsupported hash algorithm '%s' specified`, hashAlgorithm)
			}

			mode := groups[5][0]
			if mode != 'E' && mode != 'B' {
				return tail, fmt.Errorf(`Expected 'E' or 'B' as mode specifier, got '%c'`, mode)
			}

			r.Head.Version = versionNumber
			r.Head.Timestamp = timestamp
			r.Head.HashAlgorithm = hashAlgorithm
			r.Head.BasenameMode = mode == 'B'
			r.Head.NodeName = string(groups[6])
			r.Head.BasePath = string(groups[7])

			return r.Iterate() // go to next line

		} else if buffer[0] == '#' {
			// parse comment - nothing to do

		} else {
			// parse tail line
			regex, err := regexp.CompilePOSIX(`([0-9a-fA-F]+) +([A-Z]) +([0-9]+) ([^\r\n]+)`)
			if err != nil {
				return tail, err
			}

			groups := regex.FindSubmatch(buffer[0:bufferIndex])
			bytes, err := hex.DecodeString(string(groups[1]))
			if err != nil {
				return tail, fmt.Errorf(`could not decode hexdigest '%s'`, groups[1])
			}

			tail.HashValue = bytes
			tail.NodeType = groups[2][0]

			fileSize, err := strconv.Atoi(string(groups[3]))
			if err != nil {
				return tail, fmt.Errorf(`filesize is invalid: %s`, err)
			}
			tail.FileSize = uint64(fileSize)

			tail.Path = string(groups[4])
			if tail.Path == "." {
				// the external representation of the root is "."
				// the internal representation of the root is ""
				tail.Path = ""
			}
			tailLineRead = true
		}

		if tailLineRead {
			break
		}
	}

	return tail, nil
}

// Close closes the report
func (r *Report) Close() {
	if r.File != os.Stdin && r.File != os.Stdout && r.File != os.Stderr {
		r.File.Close()
	}
}

func parseVersionNumber(version string) ([3]uint16, error) {
	parts := strings.SplitN(version, ".", 3)
	var numbers [3]uint16
	for i, part := range parts {
		val, err := strconv.Atoi(part)
		if err != nil {
			return numbers, err
		}
		if val < 0 || val > 65535 {
			return numbers, fmt.Errorf(`version number specifier outside of range, 0 ≤ %d ≤ 65535 unsatisfied`, val)
		}
		numbers[i] = uint16(val)
	}
	return numbers, nil
}

func parseTimestamp(timestamp string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05", timestamp)
}

func isValidHashAlgo(hashalgo string) bool {
	whitelist := []string{
		"crc64", "crc32", "fnv-1-32", "fnv-1-64", "fnv-1-128", "fnv-1a-32", "fnv-1a-64",
		"fnv-1a-128", "adler32", "md5", "sha-1", "sha-256", "sha-512", "sha-3",
		"shake256-128",
	}
	for _, item := range whitelist {
		if item == hashalgo {
			return true
		}
	}

	return false
}