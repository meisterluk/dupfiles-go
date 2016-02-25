package utils

import (
	"crypto/sha512"
	"io"
	"log"
	"os"
	"path"

	"github.com/meisterluk/dupfiles-go/api"
)

type sha512hashing struct {
}

func copySHA512(dst []byte, src []byte) {
	for i := 0; i < sha512.Size; i++ {
		dst[i] = src[i]
	}
	for i := sha512.Size; i < api.HASHSIZE; i++ {
		dst[i] = 0
	}
}

// HashFile computes the sha512 hash of a file
func (s sha512hashing) HashFile(spec api.HashingSpec, relPath string, digest []byte) error {
	f, err := os.Open(relPath)
	if err != nil {
		return err
	}
	defer f.Close()

	hashAlgo := sha512.New()
	if spec.FileContent {
		_, err = io.Copy(hashAlgo, f)
		if err != nil {
			log.Fatal(err)
		}
	}
	if spec.FileRelPath {
		hashAlgo.Write([]byte(path.Base(relPath)))
	}

	copySHA512(digest, hashAlgo.Sum(nil))

	return nil
}

// HashString computes the sha512 hash of a string
func (s sha512hashing) HashString(hashme string, digest []byte) error {
	hashAlgo := sha512.New()
	hashAlgo.Write([]byte(hashme))

	copySHA512(digest, hashAlgo.Sum(nil))

	return nil
}

// HashTwoHashes computes the sha512 hash of two sha512 hashes
// digest and input1 or digest and input2 are allowed to overlap
func (s sha512hashing) HashTwoHashes(input1 []byte, input2 []byte, digest []byte) error {
	hashAlgo := sha512.New()
	hashAlgo.Write([]byte(input1))
	hashAlgo.Write([]byte(input2))

	copySHA512(digest, hashAlgo.Sum(nil))

	return nil
}

// HashDirectory updates the parent hash value with xor of child hash value
func (s sha512hashing) HashDirectory(parent []byte, child []byte) error {
	return xorTwoHashes(parent, child)
}

// String returns "sha512"
func (s sha512hashing) String() string {
	return "sha512"
}
