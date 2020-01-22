package internals

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

// lineToDigestFound is a simple routine to extract data from a tail line
// without instantiating an entire parser. line must not contain a line feed
// or carriage return.
func lineToDigestFound(line string, lineNo uint64) (digestFound, error) {
	i := 0

	// read hexadecimal digest
	for i < len(line) && line[i] != ' ' {
		i++
	}
	hexDigest := line[0:i]
	for i < len(line) && line[i] == ' ' {
		i++
	}

	// read node type
	if i == len(line) {
		return digestFound{}, fmt.Errorf(`line '%s' does not look like a tail line`, line)
	}
	nodeType := line[i]
	i++
	for i < len(line) && line[i] == ' ' {
		i++
	}

	// read file size
	sizeStart := i
	for i < len(line) && line[i] != ' ' {
		i++
	}
	if sizeStart == i {
		return digestFound{}, fmt.Errorf(`line '%s' does not look like a tail line`, line)
	}
	fileSize, err := strconv.ParseInt(line[sizeStart:i], 10, 64)
	if err != nil {
		return digestFound{}, fmt.Errorf(`file size is not an integer in line '%s'`, line)
	}
	i++

	// return line data
	digest, err := hex.DecodeString(hexDigest)
	if err != nil {
		return digestFound{}, fmt.Errorf(`could not decode hash value of line '%s'`, line)
	}

	return digestFound{
		0,
		lineNo,
		ReportTailLine{
			HashValue: digest,
			NodeType:  nodeType,
			FileSize:  uint64(fileSize),
			Path:      line[i:len(line)],
		},
	}, nil
}

type digestFound struct {
	ReportIndex int
	LineNo      uint64
	ReportTailLine
}

func findDigestMatchesInFile(fd *os.File, hexDigest string) ([]digestFound, error) {
	results := make([]digestFound, 0, 32)
	hexDigestIndex := 0
	lineNumber := uint64(1)

	var buffer [3072]byte

	// read from the beginning of the file
	_, err := fd.Seek(0, 0)
	if err != nil {
		return []digestFound{}, err
	}

	for {
		// reading data into buffer
		n, err := fd.Read(buffer[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return []digestFound{}, err
		}

		// search digest in buffer/view
		view := buffer[0:n]

		for offset, c := range view {
			if c == '\n' {
				hexDigestIndex = 0
				lineNumber++
			}
			if c != hexDigest[hexDigestIndex] {
				hexDigestIndex = 0
				continue
			}

			hexDigestIndex++
			if hexDigestIndex != len(hexDigest) {
				continue
			}

			hexDigestIndex = 0

			// At this point, we found the digest string in the text file.
			// So we seek to the start of the line and read tail line data.

			// seek to beginning of line
			_, err := fd.Seek(-int64(n)+int64(offset)-int64(len(hexDigest))+1, 1)
			if err != nil {
				return []digestFound{}, err
			}

			// read line
			n2, err := fd.Read(buffer[:])
			if err != nil {
				return []digestFound{}, err
			}

			line := buffer[0:n2]
			eolFound := false
			for length := 0; length < len(line); length++ {
				if line[length] == '\n' || line[length] == '\r' {
					line = buffer[0:length]
					eolFound = true
					break
				}
			}
			if !eolFound {
				// TODO not implemented yet: handle separately, do not throw error
				return []digestFound{}, fmt.Errorf(`line uses more than 2048 bytes - internal error`)
			}

			result, err := lineToDigestFound(string(line), lineNumber)
			if err != nil {
				log.Println(err)
			} else {
				results = append(results, result)
			}

			// continue reading file after this line
			_, err = fd.Seek(-int64(n2)+int64(len(line)), 1)
			if err != nil {
				return []digestFound{}, fmt.Errorf(`searching %s in %s: %s`, hexDigest, fd.Name(), err.Error())
			}
			break // must not continue inside modified view/buffer
		}
	}

	return results, nil
}

// FindDuplicates finds duplicate nodes in report files. The results are sent to outChan.
// Any errors are sent to errChan. At termination outChan and errChan are closed.
func FindDuplicates(reportFiles []string, outChan chan<- DuplicateSet, errChan chan<- error) {
	defer close(outChan)
	defer close(errChan)

	// TODO: a shortcut that determines whether the roots of all files are the same
	//   thus, start a second goroutine that checks whether all the digests “.” of the files are the same
	// TODO: a separate goroutine could read the last thirty directory digests
	//   and tell whether any are the same. If so, they could give a preinformation.
	// TODO sort digests, then lookup is much faster?
	// TODO performance improvement by some map data structures?
	// APPROACH find matches first to reduce the total amount of data
	// APPROACH create a tree of basenames, yield leaves first and if a matching node is found,
	//   bubble it up until the highest matching parent node is found.
	// APPROACH simply sorting report file lines might be faster than my implementation?

	// TODO handle files, find major parent

	// Step 0: check that parameterization is consistent
	var refVersion uint16
	var refHashAlgorithm string
	var refBasenameMode bool
	for _, reportFile := range reportFiles {
		rep, err := NewReportReader(reportFile)
		if err != nil {
			errChan <- err
			return
		}
		_, err = rep.Iterate()
		if err != nil {
			errChan <- err
			return
		}
		version := rep.Head.Version[0]
		hashAlgorithm := rep.Head.HashAlgorithm
		baseName := rep.Head.BasenameMode
		rep.Close()

		if refHashAlgorithm == "" {
			refVersion = version
			refHashAlgorithm = hashAlgorithm
			refBasenameMode = baseName
		} else {
			if refVersion != version {
				errChan <- fmt.Errorf(`Found inconsistent version among report files: %d.x as well as %d.x`, refVersion, rep.Head.Version[0])
				return
			}
			if refHashAlgorithm != hashAlgorithm {
				errChan <- fmt.Errorf(`Found inconsistent hash algorithm among report files: %s as well as %s`, refHashAlgorithm, rep.Head.HashAlgorithm)
				return
			}
			if refBasenameMode != baseName {
				errChan <- fmt.Errorf(`Found inconsistent mode among report files: basename mode as well as empty mode`)
				return
			}
		}
		// TODO: runtime.Gosched() ?
	}
	algo, err := HashAlgorithmFromString(refHashAlgorithm)
	if err != nil {
		errChan <- err
		return
	}
	digestSize := algo.DigestSize()
	basenameString := "basename"
	if !refBasenameMode {
		basenameString = "empty"
	}
	log.Printf("check for consistent metadata passed: version %d, hash algo %s, and %s mode\n", refVersion, refHashAlgorithm, basenameString)

	// Setup of a verifier that is used in Step 2
	type match struct {
		digest        string
		reportIndices []int
	}
	verifierTerminated := make(chan bool)
	verifierIsOkay := true
	toVerify := make(chan match)
	go func() {
		defer recover()

		// open all files
		reportFileDescriptors := make([]*os.File, len(reportFiles))
		for i, reportFile := range reportFiles {
			fd, e := os.Open(reportFile)
			if e != nil {
				errChan <- e
				verifierIsOkay = false
				continue
			}

			defer fd.Close()
			reportFileDescriptors[i] = fd

			// TODO: runtime.Gosched() ?
		}

		// verify all matches received of the non-closed channel
		for thisMatch := range toVerify {
			clusters := make([][]int, 0, 4)

			// file descriptors might not be properly initialized
			// we cannot proceed and expect toVerify to be closed externally soon
			if !verifierIsOkay {
				continue
			}

			// we received an unverified match.
			// we look for the hash in the report file and read it as ReportTailLine.
			// if they match for any subset of the match, we report it as verified match
			expectedLineMatches := 2 * len(thisMatch.reportIndices) // difficult to estimate
			lines := make([]digestFound, 0, expectedLineMatches)

			for reportID, reportIndex := range thisMatch.reportIndices {
				linesInReport, err := findDigestMatchesInFile(reportFileDescriptors[reportIndex], thisMatch.digest)
				if err != nil {
					errChan <- err
					verifierIsOkay = false
					continue
				}
				for i := 0; i < len(linesInReport); i++ {
					linesInReport[i].ReportIndex = reportID
					lines = append(lines, linesInReport[i])
				}
			}

			/*log.Printf(`found %d matches for %s`, len(lines), thisMatch.digest)
			for _, line := range lines {
				log.Printf(`  %s, line %d: %s`, reportFiles[line.ReportIndex], line.LineNo, line.Path)
			}*/

			// We need to cluster the given lines.
			// Two lines belong to the same cluster if they have the same (hash obviously and same) file size and node type.
			for i, tailLine := range lines {
				fitsTo := -1
			CLUSTER:
				for clusterIndex, cluster := range clusters {
					for _, index := range cluster {
						if tailLine.FileSize == lines[index].FileSize && tailLine.NodeType == lines[index].NodeType {
							fitsTo = clusterIndex
							break CLUSTER
						}
					}
				}
				if fitsTo == -1 {
					// create a new cluster
					cluster := make([]int, 0, 4)
					cluster = append(cluster, i)
					clusters = append(clusters, cluster)
					continue
				}

				// add to existing cluster
				clusters[fitsTo] = append(clusters[fitsTo], i)
			}

			//log.Println(`evaluated clusters:`, clusters)

			for _, cluster := range clusters {
				if len(cluster) <= 1 {
					// if only one item is in a cluster, it does not show equivalence to any other
					// thus, drop it
					continue
				}

				items := make([]DupOutput, 0, len(cluster))
				for _, index := range cluster {
					tailLine := lines[index]
					items = append(items, DupOutput{
						ReportFile: reportFiles[tailLine.ReportIndex],
						Lineno:     tailLine.LineNo,
						Path:       tailLine.Path,
					})
				}

				// TODO should not go to outChan, contains only non-hierarchical directory hashes
				outChan <- DuplicateSet{
					Digest: lines[0].HashValue,
					Set:    items,
				}
			}

			// TODO: runtime.Gosched() ?
		}

		verifierTerminated <- true
		return
	}()

	// Step 1: Read digests of all directories
	digests := make([]byte, 0, 256*digestSize)
	digestReportSplit := make([]int, 0, len(reportFiles))
	for _, reportFile := range reportFiles {
		// open reportFile
		rep, err := NewReportReader(reportFile)
		if err != nil {
			errChan <- err
			close(toVerify)
			return
		}

		for {
			// read each tail line
			tail, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				errChan <- err
				rep.Close()
				close(toVerify)
				return
			}

			// consider only directories
			if tail.NodeType != 'D' {
				continue
			}

			// store digest
			digests = append(digests, tail.HashValue...)
		}

		rep.Close()
		digestReportSplit = append(digestReportSplit, len(digests))

		// TODO: runtime.Gosched() ?
	}
	countDigests := len(digests) / digestSize
	log.Printf("all %d directory digests read … finding duplicates now", countDigests)

	// Step 2: Find any hashes occuring at least twice
	for a := 0; a < countDigests; a++ {
		digest := digests[int(a)*digestSize : int(a+1)*digestSize]

		// collect indices of digests that match
		occIndices := make([]uint32, 1, 32)
		occIndices[0] = uint32(a)
	SECOND:
		for b := a + 1; b < countDigests; b++ {
			for i := 0; i < digestSize; i++ {
				if digests[a*digestSize+i] != digests[b*digestSize+i] {
					continue SECOND
				}
			}
			occIndices = append(occIndices, uint32(b))
		}
		if len(occIndices) <= 1 {
			continue
		}

		// for each match, determine the index of the corresponding reportFile
		// (but store the index only once ⇒ set)
		occIndicesReportSet := make([]int, 0, len(occIndices))
		for _, index := range occIndices {
			for i, end := range digestReportSplit {
				start := 0
				if i > 0 {
					start = digestReportSplit[i-1]
				}
				if uint32(start) <= index && index < uint32(end) && (len(occIndicesReportSet) == 0 || occIndicesReportSet[len(occIndicesReportSet)-1] != i) {
					occIndicesReportSet = append(occIndicesReportSet, i)
					break
				}
			}
		}

		log.Printf(`digest %s ⇒ %d occurences among %d report files`, hex.EncodeToString(digest), len(occIndices), len(occIndicesReportSet))

		// a tiny helper: if the verifier signals it is not okay,
		// then stop sending verification tasks
		if !verifierIsOkay {
			break
		}

		// Matches show that digests are equivalent.
		// Now the verifier determines whether they are actually the same nodes (i.e. check filesize).
		// If so, it will report the result to the outChan.
		// NOTE matching is some stop-and-go operation, verifying is some stop-and-go operation.
		//      Thus, it seems intuitive to do this concurrently.
		toVerify <- match{
			digest:        hex.EncodeToString(digest),
			reportIndices: occIndicesReportSet,
		}

		// TODO: runtime.Gosched() ?
	}

	close(toVerify)
	<-verifierTerminated
}
