package internals

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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
// Returns whether excludeTree ignores this node (bool) and whether processing shall continue or not (error).
// NOTE this implementation assumes that actual directory depths do not trigger a stackoverflow (on my system, the max depth is 26, so I should be fine)
func walkDFS(nodePath string, node os.FileInfo, params *walkParameters) (bool, error) {
	// an error occured somewhere ⇒ terminated prematurely & gracefully
	if *params.shallStop {
		return true, nil
	}

	// test exclusion trees
	if contains(params.excludeTree, nodePath) {
		return false, nil
	}

	if node.IsDir() {
		fullPath := filepath.Join(params.basePath, nodePath)
		numEntries := 0
		entries, err := ioutil.ReadDir(fullPath)
		if err != nil && !(params.ignorePermErrors && isPermissionError(err)) {
			return true, err
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

			countNode, err := walkDFS(filepath.Join(nodePath, entry.Name()), entry, params)
			if err != nil {
				return true, err
			}
			if countNode {
				numEntries++
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

			countNode, err := walkDFS(filepath.Join(nodePath, entry.Name()), entry, params)
			if err != nil {
				return true, err
			}
			if countNode {
				numEntries++
			}
		}

		// TODO initialize digest with hashed basename in basename mode

		params.dirOut <- DirData{Path: nodePath, EntriesMissing: numEntries, Size: uint16(node.Size()), Digest: make([]byte, params.digestSize)}
	} else {
		params.fileOut <- FileData{Path: nodePath, Type: determineNodeType(node), Size: uint64(node.Size()), Digest: make([]byte, params.digestSize)}
	}

	runtime.Gosched() // TODO review
	return true, nil
}

// walkBFS visit all subnodes of node at nodePath in BFS manner with respect to all parameters provided.
// nodePath is relative to params.basePath. node is FileInfo of nodePath. params is uniform among all walk calls.
// Returns whether excludeTree ignores this node (bool) and whether processing shall continue or not (error).
// NOTE this implementation assumes that actual directory depths do not trigger a stackoverflow (on my system, the max depth is 26, so I should be fine)
func walkBFS(nodePath string, node os.FileInfo, params *walkParameters) (bool, error) {
	// an error occured somewhere ⇒ terminated prematurely & gracefully
	if *params.shallStop {
		return true, nil
	}

	// test exclusion trees
	if contains(params.excludeTree, nodePath) {
		return false, nil
	}

	if node.IsDir() {
		fullPath := filepath.Join(params.basePath, nodePath)
		numEntries := 0
		entries, err := ioutil.ReadDir(fullPath)
		if err != nil && !(params.ignorePermErrors && isPermissionError(err)) {
			return true, err
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

			countNode, err := walkBFS(filepath.Join(nodePath, entry.Name()), entry, params)
			if err != nil {
				return true, err
			}
			if countNode {
				numEntries++
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

			countNode, err := walkBFS(filepath.Join(nodePath, entry.Name()), entry, params)
			if err != nil {
				return true, err
			}
			if countNode {
				numEntries++
			}
		}

		// TODO initialize digest with hashed basename in basename mode

		params.dirOut <- DirData{Path: nodePath, EntriesMissing: numEntries, Size: uint16(node.Size()), Digest: make([]byte, params.digestSize)}
	} else {
		params.fileOut <- FileData{Path: nodePath, Type: determineNodeType(node), Size: uint64(node.Size()), Digest: make([]byte, params.digestSize)}
	}

	runtime.Gosched() // TODO review
	return true, nil
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
		_, err = walkDFS("", baseNodeInfo, &walkParams)
	} else {
		_, err = walkBFS("", baseNodeInfo, &walkParams)
	}
	if err != nil {
		errChan <- err
		return
	}
}

// unitHashFile computes the hash of the non-directory it receives over the inputFile channel
// and sends the annotated digest to both; unitHashDir and unitFinal.
// NOTE this function defers recover. Run it as goroutine.
func unitHashFile(hashAlgorithm hashAlgo, basenameMode bool, basePath string,
	inputFile <-chan FileData, outputDir chan<- FileData, outputFinal chan<- FileData,
	errChan chan<- error, done func(), wg *sync.WaitGroup,
) {
	defer recover()
	defer wg.Done()
	defer done()

	wg.Add(1)

	// initialize a hash instance
	hash := hashAlgorithm.Algorithm()

	// for every input, hash the file and emit it to both channels
	for fileData := range inputFile {
		fileData.Digest = hashNode(hash, basenameMode, basePath, fileData)

		outputDir <- fileData
		runtime.Gosched() // TODO review
		outputFinal <- fileData
		runtime.Gosched() // TODO review
	}
}

// unitHashDir computes hashes of directories. It receives directories from unitWalk
// (i.e. inputWalk) and receives file hashes from unitHashFile (i.e. inputFile).
// Those make up directory hashes. Once all data is collected, directory hashes
// are propagated to unitFinal (i.e. outputFinal).
// NOTE this function defers recover. Run it as goroutine.
func unitHashDir(hashAlgorithm hashAlgo,
	inputWalk <-chan DirData, inputFile <-chan FileData, outputFinal chan<- DirData,
	errChan chan<- error, wg *sync.WaitGroup,
) {
	defer recover()
	defer wg.Done()

	wg.Add(1)

	// collection of DirData with intermediate hashes
	incompleteDir := make([]DirData, 0, 100)
	var walkFinished, fileFinished bool

	// Hashes are propagated to the parent directory of a file,
	// but not more than 1 parent-level. This function is used
	// internally to propagate hashes further up.
	propagate := func(path string, digest []byte) {
		//log.Printf("propagation: node '%s'…\n", path) // TODO
		node := path
		if node == "" {
			return
		}

	PROP:
		for {
			node = dir(node)
			//log.Printf("propagation: iterate with '%s'\n", node) // TODO

			// Case 1: digest makes node complete ⇒ propagate further up
			// Case 2: node is still incomplete ⇒ stop propagation
			// Case 3: node does not exist ⇒ stop propagation, we need to wait for the actual EntriesMissing value via unitWalk

			found := false
			stop := false
			for i := 0; i < len(incompleteDir); i++ {
				if node == incompleteDir[i].Path {
					found = true
					xorByteSlices(incompleteDir[i].Digest, digest)
					incompleteDir[i].EntriesMissing--

					// emit directory hash, if all hashes were provided
					if incompleteDir[i].EntriesMissing == 0 {
						// Case 1
						digest = incompleteDir[i].Digest
						outputFinal <- incompleteDir[i]
						if i+1 >= len(incompleteDir) {
							incompleteDir = incompleteDir[:i]
						} else {
							incompleteDir = append(incompleteDir[:i], incompleteDir[i+1:]...)
						}
						if node != "" {
							//log.Printf("propagation: '%s' finished → continue propagation\n", node) // TODO
						} else {
							//log.Printf("propagation: '%s' finished → stop propagation at root node\n", node) // TODO
							break PROP
						}
					} else {
						//log.Printf("propagation: EntriesMissing of '%s' = %d …\n", incompleteDir[i].Path, incompleteDir[i].EntriesMissing) // TODO
						stop = true
					}
				}
			}

			if stop {
				//log.Printf("propagation: … '%s' is still incomplete - abort propagation\n", node)  // TODO
				break PROP // Case 2
			}

			// Case 3
			if !found {
				d := make([]byte, len(digest))
				copy(d, digest)
				incompleteDir = append(incompleteDir, DirData{
					Path: node,
					// -1 is the initial value. It will be decremented with each arriving entry.
					// Eventually the actual number of required entries is added + 1.
					// This makes EntriesMissing=0 once the digest is ready.
					EntriesMissing: -1 - 1,
					Digest:         d,
				})
				//log.Printf("propagation: entry created for '%s' - stopping propagation\n", node) // TODO
				break PROP
			}
		}
	}

LOOP:
	// terminate if unitWalk AND unitFile have terminated.
	// before that update incompleteDir until all entries are complete
	// and emit complete ones to outputFinal
	for {
		//log.Println("current state", incompleteDir) // TODO
		select {
		case dirData, ok := <-inputWalk:
			if ok {
				//log.Printf("receiving initial data for directory '%s': entries expected = %v\n", dirData.Path, dirData.EntriesMissing) // TODO
				for i := 0; i < len(incompleteDir); i++ {
					if dirData.Path == incompleteDir[i].Path {
						xorByteSlices(incompleteDir[i].Digest, dirData.Digest)
						// why "+ 1"? This is abused to distinguish value 0 from -1.
						// value EntriesMissing=0 means "all entries have been found && digest is finished".
						// value EntriesMissing=-1 means "this entry was just initialized".
						incompleteDir[i].EntriesMissing += dirData.EntriesMissing + 1
						incompleteDir[i].Size = dirData.Size
						//log.Printf("EntriesMissing of '%s' = %d (via dir)\n", incompleteDir[i].Path, incompleteDir[i].EntriesMissing) // TODO

						// emit directory hash, if all file hashes were provided
						if incompleteDir[i].EntriesMissing == 0 {
							outputFinal <- incompleteDir[i]
							digest := incompleteDir[i].Digest
							if i+1 >= len(incompleteDir) {
								incompleteDir = incompleteDir[:i]
							} else {
								incompleteDir = append(incompleteDir[:i], incompleteDir[i+1:]...)
							}
							propagate(dirData.Path, digest)
						}

						continue LOOP
					}
				}

				if dirData.EntriesMissing == 0 {
					//log.Printf("EntriesMissing of '%s' = %d (added and finished, via dir)\n", dirData.Path, dirData.EntriesMissing) // TODO
					outputFinal <- dirData
					propagate(dirData.Path, dirData.Digest)
				} else {
					//log.Printf("EntriesMissing of '%s' = %d (added, via dir)\n", dirData.Path, dirData.EntriesMissing) // TODO
					incompleteDir = append(incompleteDir, dirData)
				}
			} else {
				walkFinished = true
				if walkFinished && fileFinished {
					break LOOP
				}
			}

		case fileData, ok := <-inputFile:
			if ok {
				//log.Printf("receiving digest for file '%s'\n", fileData.Path) // TODO
				directory := dir(fileData.Path)

				for i := 0; i < len(incompleteDir); i++ {
					if directory == incompleteDir[i].Path {
						xorByteSlices(incompleteDir[i].Digest, fileData.Digest)
						incompleteDir[i].EntriesMissing--
						//log.Printf("EntriesMissing of '%s' = %d (via file)\n", incompleteDir[i].Path, incompleteDir[i].EntriesMissing) // TODO

						// emit directory hash, if all file hashes were provided
						if incompleteDir[i].EntriesMissing == 0 {
							//log.Printf("publishing '%s'\n", incompleteDir[i].Path) // TODO
							outputFinal <- incompleteDir[i]
							digest := incompleteDir[i].Digest
							if i+1 >= len(incompleteDir) {
								incompleteDir = incompleteDir[:i]
							} else {
								incompleteDir = append(incompleteDir[:i], incompleteDir[i+1:]...)
							}
							propagate(directory, digest)
						}

						continue LOOP
					}
				}

				//log.Printf("EntriesMissing of '%s' = -2 (added, via file)\n", directory) // TODO
				d := make([]byte, len(fileData.Digest))
				copy(d, fileData.Digest)
				incompleteDir = append(incompleteDir, DirData{
					Path: directory,
					// -1 is the initial value. It will be decremented with each arriving entry.
					// Eventually the actual number of required entries is added + 1.
					// This makes EntriesMissing=0 once the digest is ready.
					EntriesMissing: -1 - 1,
					Digest:         d,
				})
			} else {
				fileFinished = true
				if walkFinished && fileFinished {
					break LOOP
				}
			}
		}
		runtime.Gosched() // TODO review
	}

	// TODO verify that all entries have been emitted
	//log.Println("terminating routine. Final state:") // TODO
	//log.Println(incompleteDir) // TODO

	if len(incompleteDir) > 0 {
		errChan <- fmt.Errorf(`internal error: some directory was processed incompletely: %v`, incompleteDir)
	}

	close(outputFinal)
}

// unitFinal receives digests through the two channels inputFile and inputDir.
// It converts entries to ReportTailLines and forwards them to the outputEntry channel.
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
		runtime.Gosched() // TODO review
	}
}

// HashATree computes the hashes of all subnodes and emits them via out.
// In case of an error, one error will be written to errChan.
// Channels err and out will always be closed.
func HashATree(
	baseNode string, dfs bool, ignorePermErrors bool, hashAlgorithm string, excludeBasename, excludeBasenameRegex, excludeTree []string, basenameMode bool, concurrentFSUnits int,
	outChan chan<- ReportTailLine, errChan chan<- error,
) {
	defer close(outChan)
	defer close(errChan)

	var err error
	shallTerminate := false

	h, err := HashAlgorithmFromString(hashAlgorithm)
	if err != nil {
		errChan <- err
		return
	}

	walkToFile := make(chan FileData)
	walkToDir := make(chan DirData)
	fileToDir := make(chan FileData)
	fileToFinal := make(chan FileData)
	dirToFinal := make(chan DirData)

	workerTerminated := make(chan bool)
	errorChan := make(chan error)

	var wg sync.WaitGroup

	go unitWalk(baseNode, dfs, ignorePermErrors, excludeBasename, excludeBasenameRegex, excludeTree, h.DigestSize(), walkToFile, walkToDir, errorChan, &shallTerminate, &wg)
	for i := 0; i < 4; i++ {
		go unitHashFile(h, basenameMode, baseNode, walkToFile, fileToDir, fileToFinal, errorChan, func() {
			workerTerminated <- true
		}, &wg)
	}
	go unitHashDir(h, walkToDir, fileToDir, dirToFinal, errorChan, &wg)
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
			runtime.Gosched() // TODO review
		}
	}()

	wg.Wait()
	terminate <- true

	if err != nil {
		errChan <- err
	}
}
