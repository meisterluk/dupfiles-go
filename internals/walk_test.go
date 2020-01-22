package internals

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"hash/adler32"
	"hash/crc32"
	"hash/crc64"
	"hash/fnv"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/sha3"
)

// FILECONTENT of test file “example.txt”.
// To convert it to UTF-8, you only need to use []byte(FILECONTENT)
const FILECONTENT = `dupfiles generates rεports
😊
`

func createTestFiles(t *testing.T) string {
	base, err := ioutil.TempDir("", "dupfiles-test")
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(base, `1/3`), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(base, `1/4/7`), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	files := []string{`1/2`, `1/3/5`, `1/3/6`, `1/4/7/8`}
	for _, f := range files {
		fd, err := os.Create(filepath.Join(base, f))
		if err != nil {
			t.Fatal(err)
		}
		_, err = fd.Write([]byte(FILECONTENT))
		if err != nil {
			t.Fatal(err)
		}
		fd.Close()
	}

	return base
}

func runWalk(t *testing.T, dfs bool) []string {
	// setup
	var wg sync.WaitGroup
	digestSize := 2
	fileChan := make(chan FileData)
	dirChan := make(chan DirData)
	errChan := make(chan error)
	shallStop := false
	data := make([]string, 0, 10)

	// create test file structure temporarily
	base := createTestFiles(t)
	defer os.RemoveAll(base)

	// store the results
	wg.Add(1)
	go func() {
		defer wg.Done()
		var unitFile, unitDir bool
		for {
			select {
			case fileData, ok := <-fileChan:
				if ok {
					data = append(data, filepath.Base(fileData.Path))
				} else {
					unitFile = true
					if unitFile && unitDir {
						return
					}
				}

			case dirData, ok := <-dirChan:
				if ok {
					data = append(data, filepath.Base(dirData.Path))
				} else {
					unitDir = true
					if unitFile && unitDir {
						return
					}
				}
			}
		}
	}()

	// walk
	unitWalk(base, dfs, false, []string{}, []string{}, []string{}, digestSize, fileChan, dirChan, errChan, &shallStop, &wg)
	wg.Wait()
	go func() { close(errChan) }()
	err, ok := <-errChan
	if ok {
		t.Fatal(err)
	}

	return data
}

func TestDFSBFS(t *testing.T) {
	// run DFS
	actual := runWalk(t, true)

	// compare results
	expected := "5,6,3,8,7,4,2,1,."
	if !eqStringSlices(actual, strings.Split(expected, ",")) {
		t.Fatalf("For DFS, expected order %s got %s", expected, strings.Join(actual, ","))
	}

	// run BFS
	actual = runWalk(t, false)

	// compare results
	expected = "2,5,6,3,8,7,4,1,."
	if !eqStringSlices(actual, strings.Split(expected, ",")) {
		t.Fatalf("For BFS, expected order %s got %s", expected, strings.Join(actual, ","))
	}
}

// sha3.NewShake256() is not a hash.Hash because of its expandable output function.
// However, output length 128 is specified in the design document.
// So can make it a hash.Hash.
type shakeHash struct {
	h sha3.ShakeHash
}

func newShakeHash() shakeHash {
	return shakeHash{
		h: sha3.NewShake256(),
	}
}

func (s shakeHash) Sum(b []byte) []byte {
	sum := make([]byte, 128)
	s.h.Write(b)
	s.h.Read(sum)
	return sum
}

func (s shakeHash) Reset() {
	s.h.Reset()
}

func (s shakeHash) Size() int {
	return 16
}

func (s shakeHash) BlockSize() int {
	// SHA3 has no notion of a block size, so we return a vacuous value
	return 0
}

func (s shakeHash) Write(p []byte) (int, error) {
	return s.h.Write(p)
}

// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.

func refHash(t *testing.T, basenameMode bool, hashAlgorithm string, mode int) string {
	var h hash.Hash
	switch hashAlgorithm {
	case "crc64":
		crc64Table := crc64.MakeTable(crc64.ISO)
		h = crc64.New(crc64Table)
	case "crc32":
		crc32Table := crc32.MakeTable(crc32.IEEE)
		h = crc32.New(crc32Table)
	case "fnv-1-32":
		h = fnv.New32()
	case "fnv-1-64":
		h = fnv.New64()
	case "fnv-1-128":
		h = fnv.New128()
	case "fnv-1a-32":
		h = fnv.New32a()
	case "fnv-1a-64":
		h = fnv.New64a()
	case "fnv-1a-128":
		h = fnv.New128a()
	case "adler32":
		h = adler32.New()
	case "md5":
		h = md5.New()
	case "sha-1":
		h = sha1.New()
	case "sha-256":
		h = sha256.New()
	case "sha-512":
		h = sha512.New()
	case "sha-3":
		h = sha3.New512()
	case "shake256-128":
		h = newShakeHash()
	}

	update := func(data []byte) {
		_, err := h.Write(data)
		if err != nil {
			t.Fatal(err)
		}
	}

	switch mode {
	case 0:
		if basenameMode {
			update(append([]byte("2"), '\x1F'))
		}

		update([]byte(FILECONTENT))
		return hex.EncodeToString(h.Sum(nil))
	case 1:
		if basenameMode {
			update(append([]byte("8"), '\x1F'))
		}

		update([]byte(FILECONTENT))
		fileHashValue := h.Sum(nil)

		h.Reset()
		if basenameMode {
			update(append([]byte("7"), '\x1F'))
		}
		dirHashValue := h.Sum(nil)

		xorByteSlices(dirHashValue, fileHashValue)
		return hex.EncodeToString(dirHashValue)
	case 2:
		if basenameMode {
			update(append([]byte("5"), '\x1F'))
		}

		update([]byte(FILECONTENT))
		fileHashValue1 := h.Sum(nil)
		h.Reset()

		if basenameMode {
			update(append([]byte("6"), '\x1F'))
		}

		update([]byte(FILECONTENT))
		fileHashValue2 := h.Sum(nil)
		h.Reset()

		if basenameMode {
			update(append([]byte("3"), '\x1F'))
		}
		dirHashValue := h.Sum(nil)

		xorByteSlices(dirHashValue, fileHashValue1)
		xorByteSlices(dirHashValue, fileHashValue2)
		return hex.EncodeToString(dirHashValue)
	}

	return ""
}

func TestEvaluate(t *testing.T) {
	// create test file structure temporarily
	base := createTestFiles(t)
	defer os.RemoveAll(base)

	// define structure for reference hashes. The key structure is
	// [basenameMode][hashAlgorithm]{hash of file, hash of directory with file, hash of directory with two file, hash of .}
	refHashes := map[bool]map[string][3]string{
		true: {
			"crc64":        [3]string{"", "", ""},
			"crc32":        [3]string{"", "", ""},
			"fnv-1-32":     [3]string{"", "", ""},
			"fnv-1-64":     [3]string{"", "", ""},
			"fnv-1-128":    [3]string{"", "", ""},
			"fnv-1a-32":    [3]string{"", "", ""},
			"fnv-1a-64":    [3]string{"", "", ""},
			"fnv-1a-128":   [3]string{"", "", ""},
			"adler32":      [3]string{"", "", ""},
			"md5":          [3]string{"", "", ""},
			"sha-1":        [3]string{"", "", ""},
			"sha-256":      [3]string{"", "", ""},
			"sha-512":      [3]string{"", "", ""},
			"sha-3":        [3]string{"", "", ""},
			"shake256-128": [3]string{"", "", ""},
		},
		false: {
			"crc64":        [3]string{"", "", ""},
			"crc32":        [3]string{"", "", ""},
			"fnv-1-32":     [3]string{"", "", ""},
			"fnv-1-64":     [3]string{"", "", ""},
			"fnv-1-128":    [3]string{"", "", ""},
			"fnv-1a-32":    [3]string{"", "", ""},
			"fnv-1a-64":    [3]string{"", "", ""},
			"fnv-1a-128":   [3]string{"", "", ""},
			"adler32":      [3]string{"", "", ""},
			"md5":          [3]string{"", "", ""},
			"sha-1":        [3]string{"", "", ""},
			"sha-256":      [3]string{"", "", ""},
			"sha-512":      [3]string{"", "", ""},
			"sha-3":        [3]string{"", "", ""},
			"shake256-128": [3]string{"", "", ""},
		},
	}

	// fill with reference hashes. Storing them in source code would be okay,
	// but I have concidence that the helper function compute them accurately.
	for basenameMode, assignment := range refHashes {
		for hashAlgorithm := range assignment {
			refHashes[basenameMode][hashAlgorithm] = [3]string{
				refHash(t, basenameMode, hashAlgorithm, 0),
				refHash(t, basenameMode, hashAlgorithm, 1),
				refHash(t, basenameMode, hashAlgorithm, 2),
			}
		}
	}

	// correctness tests
	for basenameMode, assignment := range refHashes {
		for hashAlgorithm, hashes := range assignment {
			outputChan := make(chan ReportTailLine)
			errChan := make(chan error)

			go HashATree(
				base, false, false, hashAlgorithm,
				[]string{}, []string{}, []string{},
				basenameMode, 4, outputChan, errChan,
			)

			testsFinished := 0
			for entry := range outputChan {
				if entry.Path == `1/2` && entry.FileSize == 33 {
					actual := hex.EncodeToString(entry.HashValue)
					if actual != hashes[0] {
						t.Fatalf("basenameMode=%t hashAlgorithm=%s filehash is %s, expected %s", basenameMode, hashAlgorithm, actual, hashes[0])
					} else {
						testsFinished++
					}
				} else if entry.Path == `1/4/7` {
					actual := hex.EncodeToString(entry.HashValue)
					if actual != hashes[1] {
						t.Fatalf("basenameMode=%t hashAlgorithm=%s dir-with-file-hash is %s, expected %s", basenameMode, hashAlgorithm, actual, hashes[1])
					} else {
						testsFinished++
					}
				} else if entry.Path == `1/3` {
					actual := hex.EncodeToString(entry.HashValue)
					if actual != hashes[2] {
						t.Fatalf("basenameMode=%t hashAlgorithm=%s dir-with-2-files-hash is %s, expected %s", basenameMode, hashAlgorithm, actual, hashes[2])
					} else {
						testsFinished++
					}
				}
			}
			if testsFinished != 3 {
				t.Fatalf("missed some output results, passed %d of 3 tests", testsFinished)
			}

			err, ok := <-errChan
			if ok {
				t.Fatal(err)
			}
		}
	}
}

func TestEmptyHashes(t *testing.T) {
	emptyHashes := map[string]string{
		"crc32":   "68e17d95",
		"md5":     "eddc51f98f9367bffe0dec96da83648c",
		"sha-1":   "3af73983ad876cc108ef4cf7b045450a20b35780",
		"sha-256": "2f837632f54939e1824950eeaf5924e8c275a1b8443fc8bf1eab11902d185c4c",
		"sha-512": "295e43d93006798b3608170e92ac883a84eb0635be6041226ca9eda6dab7d1ab7319a59cce44187216e1fb17f94a8ec24ca6df64532765be0da0fef27a88c3f4",
	}

	base, err := ioutil.TempDir("", "dupfiles-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)

	fd, err := os.Create(filepath.Join(base, `example.txt`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = fd.Write([]byte(FILECONTENT))
	if err != nil {
		t.Fatal(err)
	}
	fd.Close()

	for hashAlgorithm, expected := range emptyHashes {
		outputChan := make(chan ReportTailLine)
		errChan := make(chan error)

		go HashATree(
			filepath.Join(base, `example.txt`), false, false, hashAlgorithm,
			[]string{}, []string{}, []string{},
			false, len(expected)/2, outputChan, errChan,
		)

		actual := ""
		for entry := range outputChan {
			actual = hex.EncodeToString(entry.HashValue)
		}

		err, ok := <-errChan
		if ok {
			t.Fatal(err)
		}

		if expected != actual {
			t.Fatalf("%s != %s", expected, actual)
		}
	}
}
