package v1

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var testFolders []string
var fileContent1 []byte
var refHashes string

func init() {
	testFolders = []string{
		`emptyfolder`,
		`onefile`,
		`twofiles`,
		`tree/sub/subsub`,
		`tree/sub2`,
	}

	fileContent1 = []byte(`dupfiles generates rεports
😊
`)
	refHashes = `algo: crc64
algo: crc32
algo: fnv-1-32
algo: fnv-1-64
algo: fnv-1-128
algo: fnv-1a-32
algo: fnv-1a-64
algo: fnv-1a-128
algo: adler32
algo: md5
algo: sha-1
algo: sha-256
algo: sha-512
algo: sha-3-512
`
}

func TestHashes(t *testing.T) {
	settings := dupfiles.Config{
		HashAlgorithm: "crc64",
	}
}

func write(path, base string, data []byte, repeat int) {
	fd, err := os.Create(filepath.Join(base, path))
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	for i := 0; i < repeat; i++ {
		_, err := fd.Write(data)
		if err != nil {
			panic(err)
		}
	}
}

func TestMain(m *testing.M) {
	var buffer [4096]byte
	for i := 0; i < 4096; i++ {
		buffer[i] = (2 * i) % 255
	}

	// setup
	base, err := ioutil.TempDir("", "dupfiles-test")
	if err != nil {
		panic(err)
	}
	for _, folder := range testFolders {
		err = os.MkdirAll(filepath.Join(base, folder), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	write(`onefile/example.txt`, base, fileContent1, 1)
	write(`twofiles/text.txt`, base, fileContent1, 1)
	write(`twofiles/bin`, base, cache[:1024], 1)
	// factor 262144 gives us a 1 GB file
	write(`tree/sub/subsub/large`, base, cache[:], 262144)
	write(`tree/sub/bin_1.txt`, base, cache[:256], 1)
	write(`tree/sub/bin_2.txt`, base, cache[:256], 1)
	write(`tree/sub2/text.txt`, base, fileContent1, 1)
	// tear down
	defer os.RemoveAll(base)

	// run tests
	os.Exit(m.Run())
}
