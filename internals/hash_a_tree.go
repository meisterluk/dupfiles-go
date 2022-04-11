package internals

import (
	"fmt"
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
	Path      string
	Type      byte
	Size      uint64
	HashValue []byte
}

// DirData contains attributes of directories
type DirData struct {
	Path           string
	EntriesMissing int
	HashValue      []byte
	Size           uint16
}

// DupOutput defines the subset of ReportTailLine which is output for a found duplicate
type DupOutput struct {
	ReportFile string
	Path       string
}

// DuplicateSet gives the set of information returned to the user if a match was found
type DuplicateSet struct {
	HashValue []byte
	Set       []DupOutput
}

// WalkParameters are parameters configuring how the walk/traversal of a file system
// shall be done.
type WalkParameters struct {
	basePath             string
	dfs                  bool
	ignorePermErrors     bool
	hashAlgorithm        string
	excludeBasename      []string
	excludeBasenameRegex []*regexp.Regexp
	excludeTree          []string
	threeMode            bool
	fileOut              chan<- FileData
	dirOut               chan<- DirData
	hashValueSize        int
	shallStop            *bool
}

// HashNode generates the hash value of a given file (at join(basePath, data.Path)).
// For directories, only the filename is hashed on three-mode.
func HashNode(hashAlgorithm HashAlgo, threeMode bool, basePath string, data FileData) Hash {
	hash := hashAlgorithm.Instance().NewCopy()
	// TODO does it make sense that basePath and data.Path is provided. We mostly need their joined version, right?

	addToHash := func(nodeType byte, path, content string) {
		if threeMode {
			// nodetype
			hash.ReadBytes([]byte{nodeType})

			// basename
			basename := filepath.Base(path)
			hash.ReadBytes([]byte(basename))
		}

		// content
		if content != "" {
			hash.ReadBytes([]byte(content))
		}
	}

	switch {
	case data.Type == 'D':
		if threeMode {
			addToHash(data.Type, data.Path, "")
			return hash.Hash()
		}
		zero := make([]byte, hash.OutputSize())
		return zero
	case data.Type == 'C':
		addToHash(data.Type, data.Path, `device file`)
		return hash.Hash()
	case data.Type == 'F':
		addToHash(data.Type, data.Path, "")
		hash.ReadFile(filepath.Join(basePath, data.Path))
		return hash.Hash()
	case data.Type == 'L':
		target, err := os.Readlink(filepath.Join(basePath, data.Path))
		if err != nil {
			return hash.Hash()
		}

		addToHash(data.Type, data.Path, `link to `)
		hash.ReadBytes([]byte(target))
		return hash.Hash()
	case data.Type == 'P':
		addToHash(data.Type, data.Path, `FIFO pipe`)
		return hash.Hash()
	case data.Type == 'S':
		addToHash(data.Type, data.Path, `UNIX domain socket`)
		return hash.Hash()
	default:
		panic(fmt.Sprintf("internal error - unknown type %c", data.Type))
	}
}

// WalkDFS visit all subnodes of node at nodePath in DFS manner with respect to all parameters provided.
// nodePath is relative to params.basePath. node is FileInfo of nodePath. params is uniform among all walk calls.
// Returns whether excludeTree ignores this node (bool) and whether processing shall continue or not (error).
// NOTE this implementation assumes that actual directory depths do not trigger a stackoverflow (on my system, the max depth is 26, so I should be fine)
func WalkDFS(nodePath string, node os.FileInfo, params *WalkParameters) (bool, error) {
	// an error occured somewhere ⇒ terminated prematurely & gracefully
	if *params.shallStop {
		return true, nil
	}

	// test exclusion trees
	if Contains(params.excludeTree, nodePath) {
		return false, nil
	}

	if node.IsDir() {
		fullPath := filepath.Join(params.basePath, nodePath)
		numEntries := 0
		entries, err := os.ReadDir(fullPath)
		if err != nil && !(params.ignorePermErrors && IsPermissionError(err)) {
			return true, err
		}

		// DFS ⇒ descend into directories immediately
	DIR:
		for _, entry := range entries {
			if !entry.IsDir() {
				continue DIR
			}

			// test exclusions
			if Contains(params.excludeBasename, entry.Name()) {
				continue DIR
			}
			for _, regex := range params.excludeBasenameRegex {
				if regex.MatchString(entry.Name()) {
					continue DIR
				}
			}

			info, err := entry.Info()
			if err != nil {
				return true, err
			}
			countNode, err := WalkDFS(filepath.Join(nodePath, entry.Name()), info, params)
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
			if Contains(params.excludeBasename, entry.Name()) {
				continue FILE
			}
			for _, regex := range params.excludeBasenameRegex {
				if regex.MatchString(entry.Name()) {
					continue FILE
				}
			}

			info, err := entry.Info()
			if err != nil {
				return true, err
			}
			countNode, err := WalkDFS(filepath.Join(nodePath, entry.Name()), info, params)
			if err != nil {
				return true, err
			}
			if countNode {
				numEntries++
			}
		}

		// in three-mode, initialize the hash value with
		// the node type and the basename concatenated and hashed
		// → hash values of children will be XORed later on
		hashValue := make(Hash, params.hashValueSize)
		if params.threeMode {
			h, err := HashAlgos{}.FromString(params.hashAlgorithm)
			if err != nil {
				return false, err
			}
			hashValue := HashNode(h, params.threeMode, filepath.Base(nodePath), FileData{
				Path:      fullPath,
				Type:      'D',
				Size:      0,
				HashValue: []byte{},
			})
			if err != nil {
				return false, err
			}
			hashValue.XOR(hashValue)
		}

		params.dirOut <- DirData{Path: nodePath, EntriesMissing: numEntries, Size: uint16(node.Size()), HashValue: hashValue}
	} else {
		params.fileOut <- FileData{Path: nodePath, Type: DetermineNodeType(node), Size: uint64(node.Size()), HashValue: make([]byte, params.hashValueSize)}
	}

	runtime.Gosched() // TODO review
	return true, nil
}

// WalkBFS visit all subnodes of node at nodePath in BFS manner with respect to all parameters provided.
// nodePath is relative to params.basePath. node is FileInfo of nodePath. params is uniform among all walk calls.
// Returns whether excludeTree ignores this node (bool) and whether processing shall continue or not (error).
// NOTE this implementation assumes that actual directory depths do not trigger a stackoverflow (on my system, the max depth is 26, so I should be fine)
func WalkBFS(nodePath string, node os.FileInfo, params *WalkParameters) (bool, error) {
	// an error occured somewhere ⇒ terminated prematurely & gracefully
	if *params.shallStop {
		return true, nil
	}

	// test exclusion trees
	if Contains(params.excludeTree, nodePath) {
		return false, nil
	}

	if node.IsDir() {
		fullPath := filepath.Join(params.basePath, nodePath)
		numEntries := 0
		entries, err := os.ReadDir(fullPath)
		if err != nil && !(params.ignorePermErrors && IsPermissionError(err)) {
			return true, err
		}

		// BFS ⇒ evaluate files first
	FILE:
		for _, entry := range entries {
			if entry.IsDir() {
				continue FILE
			}

			// test exclusions
			if Contains(params.excludeBasename, entry.Name()) {
				continue FILE
			}
			for _, regex := range params.excludeBasenameRegex {
				if regex.MatchString(entry.Name()) {
					continue FILE
				}
			}

			info, err := entry.Info()
			if err != nil {
				return true, err
			}
			countNode, err := WalkBFS(filepath.Join(nodePath, entry.Name()), info, params)
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
			if Contains(params.excludeBasename, entry.Name()) {
				continue DIR
			}
			for _, regex := range params.excludeBasenameRegex {
				if regex.MatchString(entry.Name()) {
					continue DIR
				}
			}

			info, err := entry.Info()
			if err != nil {
				return true, err
			}
			countNode, err := WalkBFS(filepath.Join(nodePath, entry.Name()), info, params)
			if err != nil {
				return true, err
			}
			if countNode {
				numEntries++
			}
		}

		// in three-mode, initialize the hash value with
		// the node type and the basename concatenated and hashed
		// → hash values of children will be XORed later on
		hashValue := make(Hash, params.hashValueSize)
		if params.threeMode {
			h, err := HashAlgos{}.FromString(params.hashAlgorithm)
			if err != nil {
				return false, err
			}
			hashValue := HashNode(h, params.threeMode, filepath.Base(nodePath), FileData{
				Path:      fullPath,
				Type:      'D',
				Size:      0,
				HashValue: []byte{},
			})
			if err != nil {
				return false, err
			}
			hashValue.XOR(hashValue)
		}

		params.dirOut <- DirData{Path: nodePath, EntriesMissing: numEntries, Size: uint16(node.Size()), HashValue: hashValue}
	} else {
		params.fileOut <- FileData{Path: nodePath, Type: DetermineNodeType(node), Size: uint64(node.Size()), HashValue: make([]byte, params.hashValueSize)}
	}

	runtime.Gosched() // TODO review
	return true, nil
}

// UnitWalk visit all subnodes of node in DFS/BFS manner with respect to all parameters provided.
// Nondirectories are emitted to fileOut. Directories are emitted to dirOut.
// If any error occurs, [only] the first error will be written to errChan. Otherwise nil is written to the error channel.
// Thus errChan also serves as signal to indicate when {fileOut, dirOut} channel won't receive any more data.
// NOTE this function defers recover. Run it as goroutine.
func UnitWalk(node string, dfs bool, ignorePermErrors bool, hashAlgorithm string, excludeBasename, excludeBasenameRegex, excludeTree []string, threeMode bool, hashValueSize int,
	fileOut chan<- FileData, dirOut chan<- DirData,
	errChan chan error, shallStop *bool, wg *sync.WaitGroup,
) {

	defer recover()
	defer wg.Done()
	defer close(fileOut)
	defer close(dirOut)

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
	walkParams := WalkParameters{
		basePath: node, dfs: dfs, ignorePermErrors: ignorePermErrors, excludeBasename: excludeBasename,
		hashAlgorithm: hashAlgorithm, excludeBasenameRegex: regexes, excludeTree: excludeTree,
		threeMode: threeMode, fileOut: fileOut, dirOut: dirOut, hashValueSize: hashValueSize,
		shallStop: shallStop,
	}

	baseNodeInfo, err := os.Stat(node)
	if err != nil {
		errChan <- err
		return
	}

	// actually traverse the file system
	if dfs {
		_, err = WalkDFS("", baseNodeInfo, &walkParams)
	} else {
		_, err = WalkBFS("", baseNodeInfo, &walkParams)
	}
	if err != nil {
		errChan <- err
		return
	}
}

// UnitHashFile computes the hash of the non-directory it receives over the inputFile channel
// and sends the annotated hash value to both; UnitHashDir and UnitFinal.
// NOTE this function defers recover. Run it as goroutine.
func UnitHashFile(hashAlgorithm HashAlgo, threeMode bool, basePath string,
	inputFile <-chan FileData, outputDir chan<- FileData, outputFinal chan<- FileData,
	errChan chan<- error, done func(), wg *sync.WaitGroup,
) {
	defer recover()
	defer wg.Done()
	defer done()

	// for every input, hash the file and emit it to both channels
	for fileData := range inputFile {
		fileData.HashValue = HashNode(hashAlgorithm, threeMode, basePath, fileData)

		outputDir <- fileData
		runtime.Gosched() // TODO review
		outputFinal <- fileData
		runtime.Gosched() // TODO review
	}
}

// UnitHashDir computes hashes of directories. It receives directories from UnitWalk
// (i.e. inputWalk) and receives file hashes from UnitHashFile (i.e. inputFile).
// Those make up directory hashes. Once all data is collected, directory hashes
// are propagated to UnitFinal (i.e. outputFinal).
// NOTE this function defers recover. Run it as goroutine.
func UnitHashDir(hashAlgorithm HashAlgo,
	inputWalk <-chan DirData, inputFile <-chan FileData, outputFinal chan<- DirData,
	errChan chan<- error, wg *sync.WaitGroup,
) {
	defer recover()
	defer wg.Done()

	// collection of DirData with intermediate hashes
	incompleteDir := make([]DirData, 0, 100)
	var walkFinished, fileFinished bool

	// Hashes are propagated to the parent directory of a file,
	// but not more than 1 parent-level. This function is used
	// internally to propagate hashes further up.
	propagate := func(path string, hashValue []byte) {
		//log.Printf("propagation: node '%s'…\n", path) // TODO
		node := path
		if node == "" {
			return
		}

	PROP:
		for {
			node = Dir(node, filepath.Separator)
			//log.Printf("propagation: iterate with '%s'\n", node) // TODO

			// Case 1: hash value makes node complete ⇒ propagate further up
			// Case 2: node is still incomplete ⇒ stop propagation
			// Case 3: node does not exist ⇒ stop propagation, we need to wait for the actual EntriesMissing value via UnitWalk

			found := false
			stop := false
			for i := 0; i < len(incompleteDir); i++ {
				if node == incompleteDir[i].Path {
					found = true
					XORByteSlices(incompleteDir[i].HashValue, hashValue)
					incompleteDir[i].EntriesMissing--

					// emit directory hash, if all hashes were provided
					if incompleteDir[i].EntriesMissing == 0 {
						// Case 1
						hashValue = incompleteDir[i].HashValue
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
				d := make([]byte, len(hashValue))
				copy(d, hashValue)
				incompleteDir = append(incompleteDir, DirData{
					Path: node,
					// -1 is the initial value. It will be decremented with each arriving entry.
					// Eventually the actual number of required entries is added + 1.
					// This makes EntriesMissing=0 once the hash value is ready.
					EntriesMissing: -1 - 1,
					HashValue:      d,
				})
				//log.Printf("propagation: entry created for '%s' - stopping propagation\n", node) // TODO
				break PROP
			}
		}
	}

LOOP:
	// terminate if UnitWalk AND unitFile have terminated.
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
						XORByteSlices(incompleteDir[i].HashValue, dirData.HashValue)
						// why "+ 1"? This is abused to distinguish value 0 from -1.
						// value EntriesMissing=0 means "all entries have been found && hash value is finished".
						// value EntriesMissing=-1 means "this entry was just initialized".
						incompleteDir[i].EntriesMissing += dirData.EntriesMissing + 1
						incompleteDir[i].Size = dirData.Size
						//log.Printf("EntriesMissing of '%s' = %d (via dir)\n", incompleteDir[i].Path, incompleteDir[i].EntriesMissing) // TODO

						// emit directory hash, if all file hashes were provided
						if incompleteDir[i].EntriesMissing == 0 {
							outputFinal <- incompleteDir[i]
							hashValue := incompleteDir[i].HashValue
							if i+1 >= len(incompleteDir) {
								incompleteDir = incompleteDir[:i]
							} else {
								incompleteDir = append(incompleteDir[:i], incompleteDir[i+1:]...)
							}
							propagate(dirData.Path, hashValue)
						}

						continue LOOP
					}
				}

				if dirData.EntriesMissing == 0 {
					//log.Printf("EntriesMissing of '%s' = %d (added and finished, via dir)\n", dirData.Path, dirData.EntriesMissing) // TODO
					outputFinal <- dirData
					propagate(dirData.Path, dirData.HashValue)
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
				//log.Printf("receiving hash value for file '%s'\n", fileData.Path) // TODO
				directory := Dir(fileData.Path, filepath.Separator)

				for i := 0; i < len(incompleteDir); i++ {
					if directory == incompleteDir[i].Path {
						XORByteSlices(incompleteDir[i].HashValue, fileData.HashValue)
						incompleteDir[i].EntriesMissing--
						//log.Printf("EntriesMissing of '%s' = %d (via file)\n", incompleteDir[i].Path, incompleteDir[i].EntriesMissing) // TODO

						// emit directory hash, if all file hashes were provided
						if incompleteDir[i].EntriesMissing == 0 {
							//log.Printf("publishing '%s'\n", incompleteDir[i].Path) // TODO
							outputFinal <- incompleteDir[i]
							hashValue := incompleteDir[i].HashValue
							if i+1 >= len(incompleteDir) {
								incompleteDir = incompleteDir[:i]
							} else {
								incompleteDir = append(incompleteDir[:i], incompleteDir[i+1:]...)
							}
							propagate(directory, hashValue)
						}

						continue LOOP
					}
				}

				//log.Printf("EntriesMissing of '%s' = -2 (added, via file)\n", directory) // TODO
				d := make([]byte, len(fileData.HashValue))
				copy(d, fileData.HashValue)
				incompleteDir = append(incompleteDir, DirData{
					Path: directory,
					// -1 is the initial value. It will be decremented with each arriving entry.
					// Eventually the actual number of required entries is added + 1.
					// This makes EntriesMissing=0 once the hash value is ready.
					EntriesMissing: -1 - 1,
					HashValue:      d,
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

// UnitFinal receives hash values through the two channels inputFile and inputDir.
// It converts entries to ReportTailLines and forwards them to the outputEntry channel.
func UnitFinal(inputFile <-chan FileData, inputDir <-chan DirData, outputEntry chan<- ReportTailLine, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	var fileFinished, dirFinished bool
LOOP:
	for {
		select {
		case fileData, ok := <-inputFile:
			if ok {
				outputEntry <- ReportTailLine{
					HashValue: fileData.HashValue,
					NodeType:  fileData.Type,
					Size:      fileData.Size,
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
					HashValue: dirData.HashValue,
					NodeType:  'D',
					Size:      uint64(dirData.Size),
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
	baseNode string, dfs bool, ignorePermErrors bool, hashAlgorithm string, excludeBasename, excludeBasenameRegex, excludeTree []string, threeMode bool, concurrentFSUnits int,
	outChan chan<- ReportTailLine, errChan chan<- error,
) {
	defer close(outChan)
	defer close(errChan)

	var err error
	shallTerminate := false

	h, err := HashAlgos{}.FromString(hashAlgorithm)
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

	// <profiling>
	/*go func() {
		time.Sleep(10 * time.Second)
		fd, err := os.Create("mem.prof")
		if err != nil {
			errChan <- err
			return
		}
		pprof.WriteHeapProfile(fd)
		fd.Close()
	}()*/
	// </profiling>

	wg.Add(3 + 4)

	go UnitWalk(baseNode, dfs, ignorePermErrors, hashAlgorithm, excludeBasename, excludeBasenameRegex, excludeTree, threeMode, h.Instance().OutputSize(), walkToFile, walkToDir, errorChan, &shallTerminate, &wg)
	for i := 0; i < 4; i++ { // TODO static number 4 is wrong, right?
		go UnitHashFile(h, threeMode, baseNode, walkToFile, fileToDir, fileToFinal, errorChan, func() {
			workerTerminated <- true
		}, &wg)
	}
	go UnitHashDir(h, walkToDir, fileToDir, dirToFinal, errorChan, &wg)
	go UnitFinal(fileToFinal, dirToFinal, outChan, errorChan, &wg)

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
