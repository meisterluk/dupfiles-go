package internals

import (
	"os"
	"time"
)

// ReportHeadLine contains data stored in the head line of a report file
type ReportHeadLine struct {
	Version       [3]uint16
	Timestamp     time.Time
	HashAlgorithm string
	BasenameMode  bool
	NodeName      string
	BasePath      string
}

// ReportTailLine contains data stored in a tail line of a report file
type ReportTailLine struct {
	HashValue []byte
	NodeType  byte
	FileSize  uint64
	Path      string
}

// Report represents a report file to be worked with (reading or writing)
type Report struct {
	File     *os.File
	FilePath string

	Head ReportHeadLine
}
