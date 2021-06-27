package v1

import (
	"fmt"

	"github.com/meisterluk/dupfiles-go/internals"
)

const VERSION_MAJOR = internals.VERSION_MAJOR
const VERSION_MINOR = internals.VERSION_MINOR
const VERSION_PATCH = internals.VERSION_PATCH
const SPEC_MAJOR = internals.SPEC_MAJOR
const SPEC_MINOR = internals.SPEC_MINOR
const SPEC_PATCH = internals.SPEC_PATCH
const RELEASE_DATE = internals.RELEASE_DATE
const LICENSE = internals.LICENSE

type Report = internals.Report

func GenerateReport(ReportParameters) error {
	return fmt.Errorf(`not implemented yet`)
}
func SupportedHashAlgorithms() []string {
	return []string{}
}
func HashOfNode(HashParameters) ([]byte, error) {
	return []byte{}, fmt.Errorf(`not implemented yet`)
}

// ComputeDigests takes all parameters for traversal and computation
// of digests, a filepath and some output channel. Then it traverses
// the given filepath and computes all digests of underlying nodes.
// Any occuring error will be reported and might lead to too few
// entries reported via the output channel.
// If filepath points to a single file, only one entry will be sent
// to the channel. If the filepath is invalid, no entry will be sent.
// The channel will be closed on finish. Thus, you can savely iterate
// over the values in a loop:
//
//    var e *error
//    go func() {
//    	  e = &df.ComputeDigests(params, "/etc", out)
//    }()
//    for entry := range out {
//        fmt.Println(entry)
//    }
//    if *e != nil {
//        log.Fatal(*e)
//    }
//
func ComputeDigests(params HashParameters, filepath string, out chan<- ReportTail) error {
	return nil
}
