package internals

import (
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

// NewReportWriter returns an freshly-initialized Report instance
func NewReportWriter(filepath string) (*Report, error) {
	report := new(Report)

	if filepath == "-" {
		report.File = os.Stdout
	} else {
		fd, err := os.Create(filepath)
		if err != nil {
			return report, err
		}
		report.File = fd
	}
	report.FilePath = filepath

	return report, nil
}

// HeadLine writes a headline to the report given the parameters provided
func (r *Report) HeadLine(hashAlgorithm string, basenameMode bool, nodeName, basePath string) error {
	mode := "E"
	if basenameMode {
		mode = "B"
	}

	_, err := fmt.Fprintf(r.File, "# 1.0.0 %s %s %s %s %s\n",
		time.Now().UTC().Format("2006-01-02T15:04:05"),
		hashAlgorithm,
		mode, nodeName, ByteEncode(basePath))
	return err
}

// TailLine writes a tailline to the report given the parameters provided
func (r *Report) TailLine(hashValue []byte, nodeType byte, fileSize uint64, path string) error {
	_, err := fmt.Fprintf(r.File, "%s %c %d %s\n",
		hex.EncodeToString(hashValue),
		nodeType, fileSize, ByteEncode(path))
	return err
}
