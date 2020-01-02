package internals

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
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
	Path string
	Type byte
	Size uint64
}

// HashedFileData extends FileData with hash information.
// TODO: maybe we want to merge FileData & HashedFileData.
type HashedFileData struct {
	FileData
	Digest []byte
}

// DirData contains attributes of directories
type DirData struct {
	Path           string
	EntriesMissing int
}

// HashedDirData extends DirData with hash information.
// TODO: maybe we want to merge DirData & HashedDirData.
type HashedDirData struct {
	Path   string
	Digest []byte
}

// HashOneNonDirectory returns (digest, file node type, file size, error)
// after hashing a node which is not a directory.
func HashOneNonDirectory(filePath string, hash Hash, basenameMode bool) ([]byte, byte, uint64, error) {
	fileInfo, err := os.Stat(filePath)
	size := uint64(fileInfo.Size())
	if err != nil {
		return []byte{}, 'X', size, err
	}

	hash.Reset()

	if basenameMode {
		hash.ReadBytes([]byte(filepath.Base(filePath)))
		hash.ReadBytes([]byte{31}) // U+001F unit separator
	}

	mode := fileInfo.Mode()
	switch {
	case mode.IsDir():
		return []byte{}, 'D', size, fmt.Errorf(`expected non-directory, got directory %s`, filePath)
	case mode&os.ModeDevice != 0: // C
		hash.ReadBytes([]byte(`device file`))
		return hash.Digest(), 'C', size, nil
	case mode.IsRegular(): // F
		hash.ReadFile(filePath)
		return hash.Digest(), 'F', size, nil
	case mode&os.ModeSymlink != 0: // L
		target, err := os.Readlink(filePath)
		if err != nil {
			return hash.Digest(), 'X', size, fmt.Errorf(`resolving FS link %s failed: %s`, filePath, err)
		}
		hash.ReadBytes([]byte(`link to `))
		hash.ReadBytes([]byte(target))
		return hash.Digest(), 'L', size, nil
	case mode&os.ModeNamedPipe != 0: // P
		hash.ReadBytes([]byte(`FIFO pipe`))
		return hash.Digest(), 'P', size, nil
	case mode&os.ModeSocket != 0: // S
		hash.ReadBytes([]byte(`UNIX domain socket`))
		return hash.Digest(), 'S', size, nil
	}
	return hash.Digest(), 'X', size, fmt.Errorf(`unknown file type at path '%s'`, filePath)
}

// HashNode generates the hash digest of a given file (at join(basePath, data.Path)).
// For directories, only the filename is hashed on basename mode.
func HashNode(hash Hash, basenameMode bool, basePath string, data FileData) []byte {
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

type walkParameters struct {
	basePath             string
	dfs                  bool
	ignorePermErrors     bool
	excludeBasename      []string
	excludeBasenameRegex []*regexp.Regexp
	excludeTree          []string
	fileOut              chan<- FileData
	dirOut               chan<- DirData
}

// WalkDFS visit all subnodes of node at nodePath in DFS manner with respect to all parameters provided.
// nodePath is relative to params.basePath. node is FileInfo of nodePath. params is uniform among all walk calls.
// NOTE this implementation assumes that actual directory depths do not trigger a stackoverflow (on my system, the max depth is 26, so I should be fine)
func WalkDFS(nodePath string, node os.FileInfo, params *walkParameters) error {
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

			if err := WalkDFS(filepath.Join(nodePath, entry.Name()), entry, params); err != nil {
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

			if err := WalkDFS(filepath.Join(nodePath, entry.Name()), entry, params); err != nil {
				return err
			}
		}

		params.dirOut <- DirData{Path: nodePath, EntriesMissing: len(entries)}
	} else {
		params.fileOut <- FileData{Path: nodePath, Type: determineNodeType(node), Size: uint64(node.Size())}
	}

	return nil
}

// WalkBFS visit all subnodes of node at nodePath in BFS manner with respect to all parameters provided.
// nodePath is relative to params.basePath. node is FileInfo of nodePath. params is uniform among all walk calls.
// Returns whether processing shall continue or not.
// NOTE this implementation assumes that actual directory depths do not trigger a stackoverflow (on my system, the max depth is 26, so I should be fine)
func WalkBFS(nodePath string, node os.FileInfo, params *walkParameters) error {
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

			if err := WalkBFS(filepath.Join(nodePath, entry.Name()), entry, params); err != nil {
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

			if err := WalkBFS(filepath.Join(nodePath, entry.Name()), entry, params); err != nil {
				return err
			}
		}

		params.dirOut <- DirData{Path: nodePath, EntriesMissing: len(entries)}
	} else {
		params.fileOut <- FileData{Path: nodePath, Type: determineNodeType(node), Size: uint64(node.Size())}
	}

	return nil
}

// WalkNode visit all subnodes of node in DFS/BFS manner with respect to all parameters provided.
// Nondirectories are emitted to fileOut. Directories are emitted to dirOut.
// If any error occurs, [only] the first error will be written to errChan. Otherwise nil is written to the error channel.
// Thus errChan also serves as signal to indicate when {fileOut, dirOut} channel won't receive any more data.
// NOTE this function is supposed to run as a goroutine.
// NOTE this function was designed to allow it to run multiple goroutines itself. Currently it does not start more goroutines.
func WalkNode(node string, dfs bool, ignorePermErrors bool, excludeBasename, excludeBasenameRegex, excludeTree []string,
	fileOut chan<- FileData, dirOut chan<- DirData, walkErrChan chan error, finish func()) {
	var err error

	// catch any potential errors
	defer recover()
	defer finish()

	// build one single params instance (to be shared among all calls)
	regexes := make([]*regexp.Regexp, 0, len(excludeBasenameRegex))
	for _, r := range excludeBasenameRegex {
		regex, e := regexp.CompilePOSIX(r)
		if e != nil {
			err = e
		}
		regexes = append(regexes, regex)
	}
	walkParams := walkParameters{
		basePath: node, dfs: dfs, ignorePermErrors: ignorePermErrors, excludeBasename: excludeBasename,
		excludeBasenameRegex: regexes, excludeTree: excludeTree, fileOut: fileOut, dirOut: dirOut,
	}

	// actually walk
	if err == nil {
		baseNodeInfo, err := os.Stat(node)
		if err == nil {
			if dfs {
				WalkDFS("", baseNodeInfo, &walkParams)
			} else {
				WalkBFS("", baseNodeInfo, &walkParams)
			}
		}
	}

	walkErrChan <- err
}

func TraverseNode(baseNode string, dfs bool, ignorePermErrors bool, hashAlgorithm string, excludeBasename, excludeBasenameRegex, excludeTree []string, basenameMode bool, concurrentFSUnits int, out chan<- ReportTailLine, errChan chan<- error) {
	var err error
	var wg sync.WaitGroup
	wg.Add(4 + concurrentFSUnits)

	// TODO if any unit emits an error, the goroutine must finish

	walkCtx, walkCancel := context.WithCancel(context.Background())
	defer walkCancel()

	// unit (1): visit all subnodes of baseNode in DFS/BFS manner
	fileChan := make(chan FileData)
	dirChan := make(chan DirData)
	walkErrChan := make(chan error)
	go WalkNode(baseNode, dfs, ignorePermErrors, excludeBasename, excludeBasenameRegex,
		excludeTree, fileChan, dirChan, walkErrChan, func() { wg.Done(); time.Sleep(time.Second); walkCancel() })

	// unit (2a): evaluate hash of non-directory ⇒ HashedFileData
	fileCtx, fileCancel := context.WithCancel(context.Background())
	defer fileCancel()

	hashedFileChan := make(chan HashedFileData)
	hashedFileErrChan := make(chan error)
	hashedFileToDirChan := make(chan HashedFileData)

	for u := 0; u < concurrentFSUnits; u++ {
		go func(unitId int) {
			defer recover()
			defer wg.Done()

			// initialize a hash instance
			hash, hashErr := HashForHashAlgo(hashAlgorithm)
			if hashErr != nil {
				hashedFileErrChan <- hashErr
				fileCancel()
				return
			}
		LOOP:
			for {
				select {
				case fileData := <-fileChan:
					// hash a file
					digest := HashNode(hash, basenameMode, baseNode, fileData)
					hfd := HashedFileData{FileData: fileData, Digest: digest}
					hashedFileChan <- hfd
					hashedFileToDirChan <- hfd
				case <-walkCtx.Done():
					break LOOP
				}
			}
		}(u)
	}

	// (2b) if it is a directory, we wait for hashes by (2a) and finally emit ⇒ HashedDirData
	dirCtx, dirCancel := context.WithCancel(context.Background())
	defer dirCancel()

	hashedDirChan := make(chan HashedDirData)

	go func() {
		defer recover()
		defer wg.Done()

		// directory hasher
		hash, e := HashForHashAlgo(hashAlgorithm)
		if err != nil {
			err = e
			return // TODO is there any further cleanup of goroutines required? tests!
		}

		// sizeof(tuple): usually 40 bytes
		type tuple struct {
			path           string
			digest         []byte
			missingEntries int
		}

		collection := make([]tuple, 0, 100)
		unitsMustFinish := 2

	LOOP:
		for {
			select {
			case filedata := <-hashedFileToDirChan:
				directory := filepath.Dir(filedata.Path)
				for i := 0; i < len(collection); i++ {
					if directory == collection[i].path {
						xorByteSlices(collection[i].digest, filedata.Digest)
						collection[i].missingEntries--

						// emit directory hash, if all entries was provided
						if collection[i].missingEntries == 0 {
							hashedDirChan <- HashedDirData{
								Path:   collection[i].path,
								Digest: collection[i].digest,
							}
						}

						continue LOOP
					}
				}
				filedata.Path = directory
				collection = append(collection, tuple{
					path:   directory,
					digest: HashNode(hash, basenameMode, baseNode, filedata.FileData),
				})
			case dirdata := <-dirChan:
				for i := 0; i < len(collection); i++ {
					if dirdata.Path == collection[i].path {
						collection[i].missingEntries += dirdata.EntriesMissing
						continue LOOP
					}
				}
				collection = append(collection, tuple{
					path:   dirdata.Path,
					digest: HashNode(hash, basenameMode, baseNode, FileData{Path: dirdata.Path, Type: 'D', Size: 0}),
				})
			case <-walkCtx.Done():
				unitsMustFinish-- // TODO this must happen atomically
				if unitsMustFinish == 0 {
					dirCancel()
					break LOOP
				}
			case <-dirCtx.Done():
				unitsMustFinish--
				if unitsMustFinish == 0 {
					dirCancel()
					break LOOP
				}
			}
		}
	}()

	// (3) {HashedFileData, HashedDirData} are combined into a final report
	terminate := make(chan bool)
	go func() {
		defer wg.Done()
		unitsMustFinish := 2

	LOOP:
		for {
			select {
			case file := <-hashedFileChan:
				out <- ReportTailLine{
					HashValue: file.Digest,
					NodeType:  file.Type,
					FileSize:  file.Size,
					Path:      file.Path,
				}
			case dir := <-hashedDirChan:
				out <- ReportTailLine{
					HashValue: dir.Digest,
					NodeType:  'D',
					FileSize:  0,
					Path:      dir.Path,
				}
			case <-fileCtx.Done():
				unitsMustFinish--
				if unitsMustFinish == 0 {
					terminate <- true
					terminate <- true
					break LOOP
				}
			case <-dirCtx.Done():
				unitsMustFinish--
				if unitsMustFinish == 0 {
					terminate <- true
					terminate <- true
					break LOOP
				}
			}
		}
	}()

	// collect all errors in one goroutine
	go func() {
		defer wg.Done()
		var e error
		for {
			select {
			case e = <-walkErrChan:
				if e != nil {
					err = e
				}
			case e = <-hashedFileErrChan:
				if e != nil {
					err = e
				}
			case <-terminate:
				return
			}
		}
	}()

	wg.Wait()
	errChan <- err
}
