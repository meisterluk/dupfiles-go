package internals

import (
	"fmt"
	"os"
	"path/filepath"
)

type Hash interface {
	// returns number of bytes of the digest
	Size() int
	// update hash state with data of file at given filepath
	ReadFile(string) error
	// update hash state with given bytes
	ReadBytes([]byte) error
	// reset hash state
	Reset()
	// get hash state digest
	Digest() []byte
	// get hash state digest represented as hexadecimal string
	HexDigest() string
	// get string representation of this hash algorithm
	HashAlgorithm() string
}

func HashForHashAlgo(hashAlgo string) (Hash, error) {
	switch hashAlgo {
	case "crc64":
		return NewCRC64(), nil
	case "crc32":
		return NewCRC32(), nil
	case "fnv-1-32":
		return NewFNV1_32(), nil
	case "fnv-1-64":
		return NewFNV1_64(), nil
	case "fnv-1-128":
		return NewFNV1_128(), nil
	case "fnv-1a-32":
		return NewFNV1a_32(), nil
	case "fnv-1a-64":
		return NewFNV1a_64(), nil
	case "fnv-1a-128":
		return NewFNV1a_128(), nil
	case "adler32":
		return NewAdler32(), nil
	case "md5":
		return NewMD5(), nil
	case "sha-1":
		return NewSHA1(), nil
	case "sha-256":
		return NewSHA256(), nil
	case "sha-512":
		return NewSHA512(), nil
	case "sha-3":
		return NewSHA3_512(), nil
	}
	return NewCRC64(), fmt.Errorf(`unknown hash algorithm '%s'`, hashAlgo)
}

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

func HashTree(baseNode string, bfs, basenameMode bool, hashAlgo string,
	excludeFilename []string, excludeFilenameRegex []string, excludeTree []string,
	out chan ReportTailLine) error {

	/*
		dirHashes := make(map[string][]byte)
		var dirHashesMutex sync.RWMutex

		var wg sync.WaitGroup
		pathChan := make(chan string, workers)
		stop := false
		var anyError error

		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go func() {
				// fetch Hash instance
				hash, err := internals.HashForHashAlgo(hashAlgo)
				if err != nil {
					anyError = err
					wg.Done()
					return
				}
				// receive paths from channels
				for path := range pathChan {
					digest, nodeType, fileSize, err := HashOneNonDirectory(path, hash, basenameMode)
					if err != nil {
						anyError = err
						wg.Done()
						return
					}
					err = rep.TailLine(digest, nodeType, fileSize, path)
					if err != nil {
						anyError = err
						wg.Done()
						return
					}
					if stop {
						break
					}
				}
				wg.Done()
			}()
		}

		err = Walk(
			baseNode,
			bfs,
			excludeFilename,
			excludeFilenameRegex,
			excludeTree,
			pathChan,
		)
		wg.Wait()
		if anyError != nil {
			handleError(anyError.Error(), 2, reportSettings.JSONOutput)
		}
		os.Exit(0)
	*/
	return nil
}
