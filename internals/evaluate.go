package internals

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
)

// This module implements the traversal logic. Concurrent units
// evaluate file-based data. How do they interact?
//
// (1) a traversal logic emits filepaths with metadata ⇒ {FileData, DirData}
// (2a) if it is a non-directory, the hash is evaluated ⇒ HashedFileData
// (2b) if it is a directory, we wait for hashes by (2a) and finally emit ⇒ HashedDirData
// (3) {HashedFileData, HashedDirData} are combined into a final report

// FileData contains attributes of non-directories
type FileData struct {
	Path   string
	Type   byte
	Size   uint64
	Digest []byte
}

// DirData contains attributes of directories
type DirData struct {
	Path           string
	EntriesMissing int
	Digest         []byte
	Size           uint16
}

// DupOutput defines the subset of ReportTailLine which is output for a found duplicate
type DupOutput struct {
	ReportFile string
	Lineno     uint64
	Path       string
}

// DuplicateSet gives the set of information returned to the user if a match was found
type DuplicateSet struct {
	Digest []byte
	Set    []DupOutput
}

type walkParameters struct {
	basePath             string
	dfs                  bool
	ignorePermErrors     bool
	excludeBasename      []string
	excludeBasenameRegex []*regexp.Regexp
	excludeTree          []string
	fileOut              chan<- FileData
	dirOut               chan<- DirData
	digestSize           int
	shallStop            *bool
}

// hashNode generates the hash digest of a given file (at join(basePath, data.Path)).
// For directories, only the filename is hashed on basename mode.
func hashNode(hash Hash, basenameMode bool, basePath string, data FileData) []byte {
	hash.Reset()

	if basenameMode {
		hash.ReadBytes([]byte(filepath.Base(data.Path)))
		hash.ReadBytes([]byte{31}) // U+001F unit separator
	}

	switch {
	case data.Type == 'D':
		return hash.Digest()
	case data.Type == 'C':
		hash.ReadBytes([]byte(`device file`))
		return hash.Digest()
	case data.Type == 'F':
		hash.ReadFile(filepath.Join(basePath, data.Path))
		return hash.Digest()
	case data.Type == 'L':
		target, err := os.Readlink(filepath.Join(basePath, data.Path))
		if err != nil {
			return hash.Digest()
		}
		hash.ReadBytes([]byte(`link to `))
		hash.ReadBytes([]byte(target))
		return hash.Digest()
	case data.Type == 'P':
		hash.ReadBytes([]byte(`FIFO pipe`))
		return hash.Digest()
	case data.Type == 'S':
		hash.ReadBytes([]byte(`UNIX domain socket`))
		return hash.Digest()
	default:
		panic(fmt.Sprintf("internal error - unknown type %c", data.Type))
	}
}

// walkDFS visit all subnodes of node at nodePath in DFS manner with respect to all parameters provided.
// nodePath is relative to params.basePath. node is FileInfo of nodePath. params is uniform among all walk calls.
// NOTE this implementation assumes that actual directory depths do not trigger a stackoverflow (on my system, the max depth is 26, so I should be fine)
func walkDFS(nodePath string, node os.FileInfo, params *walkParameters) error {
	// an error occured somewhere ⇒ terminated prematurely & gracefully
	if *params.shallStop {
		return nil
	}

	// test exclusion trees
	if contains(params.excludeTree, nodePath) {
		return nil
	}

	if node.IsDir() {
		fullPath := filepath.Join(params.basePath, nodePath)
		entries, err := ioutil.ReadDir(fullPath)
		if err != nil && !(params.ignorePermErrors && isPermissionError(err)) {
			return err
		}

		// DFS ⇒ descend into directories immediately
	DIR:
		for _, entry := range entries {
			if !entry.IsDir() {
				continue DIR
			}

			// test exclusions
			if contains(params.excludeBasename, entry.Name()) {
				continue DIR
			}
			for _, regex := range params.excludeBasenameRegex {
				if regex.MatchString(entry.Name()) {
					continue DIR
				}
			}

			if err := walkDFS(filepath.Join(nodePath, entry.Name()), entry, params); err != nil {
				return err
			}
		}

		// … and finally all files
	FILE:
		for _, entry := range entries {
			if entry.IsDir() {
				continue FILE
			}

			// test exclusions
			if contains(params.excludeBasename, entry.Name()) {
				continue FILE
			}
			for _, regex := range params.excludeBasenameRegex {
				if regex.MatchString(entry.Name()) {
					continue FILE
				}
			}

			if err := walkDFS(filepath.Join(nodePath, entry.Name()), entry, params); err != nil {
				return err
			}
		}

		params.dirOut <- DirData{Path: nodePath, EntriesMissing: len(entries), Size: uint16(node.Size()), Digest: make([]byte, params.digestSize)}
	} else {
		params.fileOut <- FileData{Path: nodePath, Type: determineNodeType(node), Size: uint64(node.Size()), Digest: make([]byte, params.digestSize)}
	}

	return nil
}

// walkBFS visit all subnodes of node at nodePath in BFS manner with respect to all parameters provided.
// nodePath is relative to params.basePath. node is FileInfo of nodePath. params is uniform among all walk calls.
// Returns whether processing shall continue or not.
// NOTE this implementation assumes that actual directory depths do not trigger a stackoverflow (on my system, the max depth is 26, so I should be fine)
func walkBFS(nodePath string, node os.FileInfo, params *walkParameters) error {
	// an error occured somewhere ⇒ terminated prematurely & gracefully
	if *params.shallStop {
		return nil
	}

	// test exclusion trees
	if contains(params.excludeTree, nodePath) {
		return nil
	}

	if node.IsDir() {
		fullPath := filepath.Join(params.basePath, nodePath)
		entries, err := ioutil.ReadDir(fullPath)
		if err != nil && !(params.ignorePermErrors && isPermissionError(err)) {
			return err
		}

		// BFS ⇒ evaluate files first
	FILE:
		for _, entry := range entries {
			if entry.IsDir() {
				continue FILE
			}

			// test exclusions
			if contains(params.excludeBasename, entry.Name()) {
				continue FILE
			}
			for _, regex := range params.excludeBasenameRegex {
				if regex.MatchString(entry.Name()) {
					continue FILE
				}
			}

			if err := walkBFS(filepath.Join(nodePath, entry.Name()), entry, params); err != nil {
				return err
			}
		}

		// … and finally descend into directories
	DIR:
		for _, entry := range entries {
			if !entry.IsDir() {
				continue DIR
			}

			// test exclusions
			if contains(params.excludeBasename, entry.Name()) {
				continue DIR
			}
			for _, regex := range params.excludeBasenameRegex {
				if regex.MatchString(entry.Name()) {
					continue DIR
				}
			}

			if err := walkBFS(filepath.Join(nodePath, entry.Name()), entry, params); err != nil {
				return err
			}
		}

		params.dirOut <- DirData{Path: nodePath, EntriesMissing: len(entries), Size: uint16(node.Size()), Digest: make([]byte, params.digestSize)}
	} else {
		params.fileOut <- FileData{Path: nodePath, Type: determineNodeType(node), Size: uint64(node.Size()), Digest: make([]byte, params.digestSize)}
	}

	return nil
}

// unitWalk visit all subnodes of node in DFS/BFS manner with respect to all parameters provided.
// Nondirectories are emitted to fileOut. Directories are emitted to dirOut.
// If any error occurs, [only] the first error will be written to errChan. Otherwise nil is written to the error channel.
// Thus errChan also serves as signal to indicate when {fileOut, dirOut} channel won't receive any more data.
// NOTE this function defers recover. Run it as goroutine.
func unitWalk(node string, dfs bool, ignorePermErrors bool, excludeBasename, excludeBasenameRegex, excludeTree []string, digestSize int,
	fileOut chan<- FileData, dirOut chan<- DirData,
	errChan chan error, shallStop *bool, wg *sync.WaitGroup,
) {

	defer recover()
	defer wg.Done()
	defer close(fileOut)
	defer close(dirOut)

	wg.Add(1)

	// build one single params instance (to be shared among all recursive call)
	regexes := make([]*regexp.Regexp, 0, len(excludeBasenameRegex))
	for _, r := range excludeBasenameRegex {
		regex, e := regexp.CompilePOSIX(r)
		if e != nil {
			errChan <- e
			return
		}
		regexes = append(regexes, regex)
	}
	walkParams := walkParameters{
		basePath: node, dfs: dfs, ignorePermErrors: ignorePermErrors, excludeBasename: excludeBasename,
		excludeBasenameRegex: regexes, excludeTree: excludeTree, fileOut: fileOut, dirOut: dirOut,
		digestSize: digestSize, shallStop: shallStop,
	}

	baseNodeInfo, err := os.Stat(node)
	if err != nil {
		errChan <- err
		return
	}
	// actually traverse the file system
	if dfs {
		err = walkDFS(".", baseNodeInfo, &walkParams)
	} else {
		err = walkBFS(".", baseNodeInfo, &walkParams)
	}
	if err != nil {
		errChan <- err
		return
	}
}

// unitHashFile computes the hash of the non-directory it receives over the inputFile channel
// and sends the annotated digest to both; unitHashDir and unitFinal.
// NOTE this function defers recover. Run it as goroutine.
func unitHashFile(hashAlgorithm string, basenameMode bool, basePath string,
	inputFile <-chan FileData, outputDir chan<- FileData, outputFinal chan<- FileData,
	errChan chan<- error, done func(), wg *sync.WaitGroup,
) {
	defer recover()
	defer wg.Done()
	defer done()

	wg.Add(1)

	// initialize a hash instance
	hash, err := HashForHashAlgo(hashAlgorithm)
	if err != nil {
		errChan <- err
		return
	}

	// for every input, hash the file and emit it to both channels
	for fileData := range inputFile {
		fileData.Digest = hashNode(hash, basenameMode, basePath, fileData)

		outputDir <- fileData
		outputFinal <- fileData
	}
}

// unitHashDir computes hashes of directories. It receives directories from unitWalk
// (i.e. inputWalk) and receives file hashes from unitHashFile (i.e. inputFile).
// Those make up directory hashes. Once all data is collected, directory hashes
// are propagated to unitFinal (i.e. outputFinal).
// NOTE this function defers recover. Run it as goroutine.
func unitHashDir(hashAlgorithm string,
	inputWalk <-chan DirData, inputFile <-chan FileData, outputFinal chan<- DirData,
	errChan chan<- error, wg *sync.WaitGroup,
) {
	defer recover()
	defer wg.Done()

	wg.Add(1)

	// collection of DirData with intermediate hashes
	incompleteDir := make([]DirData, 0, 100)
	var walkFinished, fileFinished bool

LOOP:
	// terminate if unitWalk AND unitFile have terminated.
	// before that update incompleteDir until all entries are complete
	// and emit complete ones it outputFinal
	for {
		select {
		case dirData, ok := <-inputWalk:
			if ok {
				for i := 0; i < len(incompleteDir); i++ {
					if dirData.Path == incompleteDir[i].Path {
						xorByteSlices(incompleteDir[i].Digest, dirData.Digest)
						incompleteDir[i].EntriesMissing += dirData.EntriesMissing
						incompleteDir[i].Size = dirData.Size

						// emit directory hash, if all file hashes were provided
						if incompleteDir[i].EntriesMissing == 0 {
							outputFinal <- incompleteDir[i]
							incompleteDir = append(incompleteDir[:i], incompleteDir[i+1:]...)
						}

						continue LOOP
					}
				}
				incompleteDir = append(incompleteDir, dirData)
			} else {
				walkFinished = true
				if walkFinished && fileFinished {
					break LOOP
				}
			}

		case fileData, ok := <-inputFile:
			if ok {
				directory := filepath.Dir(fileData.Path)

				for i := 0; i < len(incompleteDir); i++ {
					if directory == incompleteDir[i].Path {
						xorByteSlices(incompleteDir[i].Digest, fileData.Digest)
						incompleteDir[i].EntriesMissing--

						// emit directory hash, if all file hashes were provided
						if incompleteDir[i].EntriesMissing == 0 {
							outputFinal <- incompleteDir[i]
							incompleteDir = append(incompleteDir[:i], incompleteDir[i+1:]...)
						}

						continue LOOP
					}
				}

				incompleteDir = append(incompleteDir, DirData{
					Path:           directory,
					EntriesMissing: -1, // this is the initial value. It will be decremented until it will become 0 again
					Digest:         fileData.Digest,
				})
			} else {
				fileFinished = true
				if walkFinished && fileFinished {
					break LOOP
				}
			}
		}
	}

	close(outputFinal)
}

func unitFinal(inputFile <-chan FileData, inputDir <-chan DirData, outputEntry chan<- ReportTailLine, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	wg.Add(1)

	var fileFinished, dirFinished bool
LOOP:
	for {
		select {
		case fileData, ok := <-inputFile:
			if ok {
				outputEntry <- ReportTailLine{
					HashValue: fileData.Digest,
					NodeType:  fileData.Type,
					FileSize:  fileData.Size,
					Path:      fileData.Path,
				}
			} else {
				fileFinished = true
				if fileFinished && dirFinished {
					break LOOP
				}
			}

		case dirData, ok := <-inputDir:
			if ok {
				outputEntry <- ReportTailLine{
					HashValue: dirData.Digest,
					NodeType:  'D',
					FileSize:  uint64(dirData.Size),
					Path:      dirData.Path,
				}
			} else {
				dirFinished = true
				if fileFinished && dirFinished {
					break LOOP
				}
			}
		}
	}
}

// Evaluate computes the hashes of all subnodes and emits them via out.
// In case of an error, one error will be written to errChan.
// Channels err and out will always be closed.
func Evaluate(
	baseNode string, dfs bool, ignorePermErrors bool, hashAlgorithm string, excludeBasename, excludeBasenameRegex, excludeTree []string, basenameMode bool, concurrentFSUnits int,
	outChan chan<- ReportTailLine, errChan chan<- error,
) {
	defer close(outChan)
	defer close(errChan)

	var err error
	shallTerminate := false

	walkToFile := make(chan FileData)
	walkToDir := make(chan DirData)
	fileToDir := make(chan FileData)
	fileToFinal := make(chan FileData)
	dirToFinal := make(chan DirData)

	workerTerminated := make(chan bool)
	errorChan := make(chan error)

	var wg sync.WaitGroup

	go unitWalk(baseNode, dfs, ignorePermErrors, excludeBasename, excludeBasenameRegex, excludeTree, OutputSizeForHashAlgo(hashAlgorithm), walkToFile, walkToDir, errorChan, &shallTerminate, &wg)
	for i := 0; i < 4; i++ {
		go unitHashFile(hashAlgorithm, basenameMode, baseNode, walkToFile, fileToDir, fileToFinal, errorChan, func() {
			workerTerminated <- true
		}, &wg)
	}
	go unitHashDir(hashAlgorithm, walkToDir, fileToDir, dirToFinal, errorChan, &wg)
	go unitFinal(fileToFinal, dirToFinal, outChan, errorChan, &wg)

	// worker counting goroutine
	wg.Add(1)
	go func() {
		// we close these channels only once all workers have terminated
		defer wg.Done()
		for i := 0; i < 4; i++ {
			<-workerTerminated
		}
		close(fileToDir)
		close(fileToFinal)
	}()

	// error handling goroutine
	terminate := make(chan bool)
	go func() {
	LOOP:
		for {
			select {
			case <-terminate:
				break LOOP
			case e := <-errorChan:
				shallTerminate = true
				err = e
			}
		}
	}()

	wg.Wait()
	terminate <- true

	if err != nil {
		errChan <- err
	}
}

// lineToDigestFound is a simple routine to parse a line into a tail line
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

	// return ReportTailLine
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

	var buffer [2048]byte

	log.Printf(`<find digest %s in file %s>`, hexDigest, fd.Name())

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

			currentPosition, err := fd.Seek(0, 1)
			if err != nil {
				panic(err)
				return []digestFound{}, err
			}
			continueAtPosition := currentPosition - int64(n) + int64(offset)
			startOfLine := continueAtPosition - int64(len(hexDigest)) + 1

			_, err = fd.Seek(int64(startOfLine), 0)
			if err != nil {
				panic(err)
				return []digestFound{}, err
			}

			n2, err := fd.Read(buffer[:])
			if err != nil {
				panic(err)
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

			log.Printf(`line is given by '%v' at %d`, string(line), startOfLine)

			result, err := lineToDigestFound(string(line), lineNumber)
			/*if err != nil {
				log.Println(`<debug>`)
				log.Printf(`digest?? %q`, hexDigest)
				log.Printf(`view?? looking for %q in '%v'`, hexDigest, string(view))
				log.Printf(`position of line? %d - %d + %d + 1 - %d = %d`, currentPosition, int64(n), int64(offset), int64(len(hexDigest)), startOfLine)
				log.Printf(`line??? '%v'`, string(line))
				log.Printf(`continueAtPosition = %d - %d + %d + 1 = %d; startOfLine = %d`, currentPosition, int64(n), int64(offset), continueAtPosition, startOfLine)
				log.Printf(`continue at %d after %d, looking for %s`, continueAtPosition, currentPosition, hexDigest)
				log.Println(`</debug>`)
				return []digestFound{}, err
			}*/
			results = append(results, result)

			// continue reading file where we left off
			_, err = fd.Seek(continueAtPosition, 0)
			if err != nil {
				panic(err)
				return []digestFound{}, err
			}
		}
	}

	log.Printf(`</find digest %s in file %s>`, hexDigest, fd.Name())
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
	var refHashAlgo string
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
		hashAlgo := rep.Head.HashAlgorithm
		baseName := rep.Head.BasenameMode
		rep.Close()

		if refHashAlgo == "" {
			refVersion = version
			refHashAlgo = hashAlgo
			refBasenameMode = baseName
		} else {
			if refVersion != version {
				errChan <- fmt.Errorf(`Found inconsistent version among report files: %d.x as well as %d.x`, refVersion, rep.Head.Version[0])
				return
			}
			if refHashAlgo != hashAlgo {
				errChan <- fmt.Errorf(`Found inconsistent hashAlgo among report files: %s as well as %s`, refHashAlgo, rep.Head.HashAlgorithm)
				return
			}
			if refBasenameMode != baseName {
				errChan <- fmt.Errorf(`Found inconsistent mode among report files: basename mode as well as empty mode`)
				return
			}
		}
	}
	digestSize := OutputSizeForHashAlgo(refHashAlgo)
	basenameString := "basename"
	if !refBasenameMode {
		basenameString = "empty"
	}
	log.Printf("check for consistent metadata passed: version %d, hash algo %s, and %s mode\n", refVersion, refHashAlgo, basenameString)

	// Setup of a verifier that is used in Step 2
	type match struct {
		digest        string
		reportIndices []int
	}
	verifierTerminated := make(chan bool)
	verifierIsOkay := true
	toVerify := make(chan match)
	finalResults := make(chan DuplicateSet)
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
		}

		// verify all matches received of the non-closed channel
		for thisMatch := range toVerify {
			clusters := make([][]int, 0, 4)

			// file descriptors might not be properly initialized
			// we cannot proceed and expect toVerify to be closed externally soon
			if !verifierIsOkay {
				continue
			}

			log.Printf(`<Verifying matches of digest %s>`, thisMatch.digest)

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

			log.Printf(`found %d matches for %s`, len(lines), thisMatch.digest)
			for _, line := range lines {
				log.Printf(`  %s, line %d: %s`, reportFiles[line.ReportIndex], line.LineNo, line.Path)
			}

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

			log.Println(`evaluated clusters:`, clusters)

			for _, cluster := range clusters {
				if len(clusters) <= 1 {
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

				finalResults <- DuplicateSet{
					Digest: lines[0].HashValue,
					Set:    items,
				}
			}
			log.Printf(`</Verifying matches of digest %s>`, thisMatch.digest)
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
	}

	close(toVerify)
	<-verifierTerminated
}

func Find_Duplicates(reportFiles []string, outChan chan<- ReportTailLine, errChan chan<- error) {
	defer close(outChan)
	defer close(errChan)

	// Step 0: check that parameterization is consistent
	init := false
	var refVersion uint16
	var refHashAlgo string
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
		if !init {
			refVersion = rep.Head.Version[0]
			refHashAlgo = rep.Head.HashAlgorithm
			refBasenameMode = rep.Head.BasenameMode
		} else {
			if refVersion != rep.Head.Version[0] {
				errChan <- fmt.Errorf(`Found inconsistent version among report files: %d as well as %d`, refVersion, rep.Head.Version[0])
				rep.Close()
				return
			}
			if refHashAlgo != rep.Head.HashAlgorithm {
				errChan <- fmt.Errorf(`Found inconsistent hashAlgo among report files: %s as well as %s`, refHashAlgo, rep.Head.HashAlgorithm)
				rep.Close()
				return
			}
			if refBasenameMode != rep.Head.BasenameMode {
				errChan <- fmt.Errorf(`Found inconsistent mode among report files: basename mode as well as empty mode`)
				rep.Close()
				return
			}
		}
		rep.Close()
	}
	digestSize := OutputSizeForHashAlgo(refHashAlgo)

	// Step 1: find all hashes occuring at least twice
	type match struct {
		digest  []byte
		reports []int
	}
	matches := make([]match, 0, 512)

	// One “master” reportfile emits hashes,
	// several “seeker”s respond how of this hash occurs in their report files
	for _, mFile := range reportFiles {
		var wg2 sync.WaitGroup
		wg2.Add(1 + len(reportFiles))

		masterChans := make([]chan []byte, len(reportFiles))
		seekerChan := make(chan int) // -1 = done, n = reportFiles[n] contains hash

		for i := 0; i < len(reportFiles); i++ {
			masterChans[i] = make(chan []byte)
		}

		// master goroutine
		go func(masterFile string, countSeekers int) {
			defer wg2.Done()
			defer func() {
				for i := 0; i < countSeekers; i++ {
					close(masterChans[i])
				}
			}()

			rep, err := NewReportReader(masterFile)
			if err != nil {
				errChan <- err
				return
			}
			defer rep.Close()

			for {
				tail, err := rep.Iterate()
				if err == io.EOF {
					break
				}
				if err != nil {
					errChan <- err
					return
				}

				// known hashes can be skipped
				wasTested := false
				for _, m := range matches {
					if compareBytes(tail.HashValue, m.digest) {
						wasTested = true
						break
					}
				}
				if wasTested {
					continue
				}

				for i := 0; i < countSeekers; i++ {
					log.Println("distributing hash to channel", i, tail.HashValue)
					masterChans[i] <- tail.HashValue
					log.Println("finished distribution to channel", i)
				}
				log.Println("finished digest distribution")

				indices := make([]int, 0, countSeekers+1)
			SEEKER:
				for i := 0; i < countSeekers; i++ {
					for {
						log.Println("waiting for result")
						result := <-seekerChan
						if result == -1 {
							continue SEEKER
						} else {
							indices = append(indices, result)
						}
					}
				}

				if len(indices) > 1 {
					matches = append(matches, match{
						digest:  tail.HashValue,
						reports: indices,
					})
				}
			}
		}(mFile, len(reportFiles))

		for sID, sFile := range reportFiles {
			// seeker goroutines
			go func(seekFile string, seekerID int) {
				log.Println("starting seeker", seekerID)
				defer wg2.Done()

				hashes := make([]byte, 0, 250*digestSize)

				// (1) read all digests to hashes
				rep, err := NewReportReader(seekFile)
				if err != nil {
					errChan <- err
					return
				}
				defer rep.Close()

				for {
					tail, err := rep.Iterate()
					if err == io.EOF {
						log.Println("seek file EOF")
						break
					}
					if err != nil {
						log.Println("error", err)
						errChan <- err
						return
					}

					for i := 0; i < digestSize; i++ {
						hashes = append(hashes, tail.HashValue[i])
					}
				}
				log.Println("finished reading hashes")

				// (2) expect some digest via masterChan
				for {
					log.Println("waiting for digest", seekerID)
					masterDigest, ok := <-masterChans[seekerID]
					log.Println("digest received", seekerID, ok, masterDigest)
					if !ok {
						log.Println("seeker terminated")
						break
					}

					log.Println("searching")
					for i := 0; i < len(hashes)/digestSize; i++ {
						j := uint64(i)
						log.Println("sending result")
						if compareBytes(masterDigest, hashes[j*uint64(digestSize):(j+1)*uint64(digestSize)]) {
							seekerChan <- seekerID
						}
					}

					seekerChan <- -1
				}
			}(sFile, sID)
		}

		wg2.Wait()
	}

	for _, m := range matches {
		log.Printf("%s %v\n", hex.EncodeToString(m.digest), m.reports)
	}

	// Step 2: limit hashes to ones which are “truly” equivalent - mainly by filesize

	// Step 3: determine most generic nodes sharing hashes
	//   TODO write nil to errChan
}
