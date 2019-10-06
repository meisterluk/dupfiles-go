package internals

import (
	"os"
	"time"
)

type ReportHeadLine struct {
	Version       [3]uint16
	Timestamp     time.Time
	HashAlgorithm string
	BasenameMode  bool
	NodeName      string
	BasePath      string
}

type ReportTailLine struct {
	HashValue []byte
	NodeType  byte
	FileSize  uint64
	Path      string
}

type Report struct {
	File     *os.File
	FilePath string

	Head ReportHeadLine
}
