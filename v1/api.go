package v1

const VERSION [3]int = [3]int{1, 0, 0}

func GenerateReport(ReportParameters) error {}
func ReadReport(path string) (ReportHead, []ReportTail, error) {}
func WriteReport(path string, ReportHead, []ReportTail) error {}
func SupportedHashAlgorithms() []string {}
func HashOfNode(HashParameters) ([]byte, error)
//func TraverseNode(HashParameters, chan<- ReportTail) error