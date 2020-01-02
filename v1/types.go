package v1

import "github.com/meisterluk/dupfiles-go/internals"

type ReportHead = internals.ReportHeadLine
type ReportTail = internals.ReportTailLine

type ReportParameters struct {
}

type HashParameters struct {
	BaseNode             string
	BFS                  bool
	DFS                  bool
	IgnorePermErrors     bool
	HashAlgorithm        string
	ExcludeBasename      []string
	ExcludeBasenameRegex []string
	ExcludeTree          []string
	BasenameMode         bool
	EmptyMode            bool
	Workers              int
}
