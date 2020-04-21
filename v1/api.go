package v1

import (
	"fmt"
)

const VERSION_MAJOR = 1
const VERSION_MINOR = 0
const VERSION_PATCH = 0
const RELEASE_DATE = "2020-04-29"

func GenerateReport(ReportParameters) error {
	return fmt.Errorf(`not implemented yet`)
}
func ReadReport(path string) (ReportHead, []ReportTail, error) {
	return ReportHead{}, make([]ReportTail, 0), fmt.Errorf(`not implemented yet`)
}
func WriteReport(path string, head ReportHead, tail []ReportTail) error {
	return fmt.Errorf(`not implemented yet`)
}
func SupportedHashAlgorithms() []string {
	return []string{}
}
func HashOfNode(HashParameters) ([]byte, error) {
	return []byte{}, fmt.Errorf(`not implemented yet`)
}

//func TraverseNode(HashParameters, chan<- ReportTail) error
