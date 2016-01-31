package utils

import (
	"crypto/sha256"
	"io"
	"log"
	"os"
	"path"

	"github.com/meisterluk/dupfiles-go/api"
)

type sha256hashing struct {
}

func copySHA256(dst []byte, src []byte) {
	for i := 0; i < sha256.Size; i++ {
		dst[i] = src[i]
	}
	for i := sha256.Size; i < api.HASHSIZE; i++ {
		dst[i] = 0
	}
}

// HashFile computes the sha256 hash of a file
func (s sha256hashing) HashFile(spec api.HashingSpec, relPath string, digest []byte) error {
	f, err := os.Open(relPath)
	if err != nil {
		return err
	}
	defer f.Close()

	hashAlgo := sha256.New()
	if spec.Content {
		_, err = io.Copy(hashAlgo, f)
		if err != nil {
			log.Fatal(err)
		}
	}
	if spec.Relpath {
		hashAlgo.Write([]byte(path.Base(relPath)))
	}

	copySHA256(digest, hashAlgo.Sum(nil))

	return nil
}

// HashString computes the sha256 hash of a string
func (s sha256hashing) HashString(hashme string, digest []byte) error {
	hashAlgo := sha256.New()
	hashAlgo.Write([]byte(hashme))

	copySHA256(digest, hashAlgo.Sum(nil))

	return nil
}

// HashTwoHashes computes the sha256 hash of two sha256 hashes
// digest and input1 or digest and input2 are allowed to overlap
func (s sha256hashing) HashTwoHashes(input1 []byte, input2 []byte, digest []byte) error {
	hashAlgo := sha256.New()
	hashAlgo.Write([]byte(input1))
	hashAlgo.Write([]byte(input2))

	copySHA256(digest, hashAlgo.Sum(nil))

	return nil
}

// HashDirectory updates the parent hash value with xor of child hash value
func (s sha256hashing) HashDirectory(parent []byte, child []byte) error {
	return xorTwoHashes(parent, child)
}

// String returns "sha256"
func (s sha256hashing) String() string {
	return "sha256"
}
