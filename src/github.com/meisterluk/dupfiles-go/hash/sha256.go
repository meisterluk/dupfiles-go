package hash

import (
	"crypto/sha256"
	"io"
	"log"
	"os"
	"path"
)

// SHA256FileHash compute the sha256 hash of a file
func SHA256FileHash(digest []byte, relPath string) error {
	f, err := os.Open(relPath)
	if err != nil {
		return err
	}
	defer f.Close()

	hashAlgo := sha256.New()
	_, err = io.Copy(hashAlgo, f)
	if err != nil {
		log.Fatal(err)
	}
	hashAlgo.Write([]byte(path.Base(relPath)))

	dig := hashAlgo.Sum(nil)
	for i := 0; i < sha256.Size; i++ {
		digest[i] = dig[i]
	}
	return nil
}

// SHA256String computes the sha256 hash of a string
func SHA256String(digest []byte, hashme string) error {
	hashAlgo := sha256.New()
	hashAlgo.Write([]byte(hashme))
	dig := hashAlgo.Sum(nil)
	for i := 0; i < sha256.Size; i++ {
		digest[i] = dig[i]
	}
	return nil
}

// SHA256HashTwoHashes computes the sha256 hash of two sha256 hashes
// digest and input1 or digest and input2 are allowed to overlap
func SHA256HashTwoHashes(digest []byte, input1 []byte, input2 []byte) error {
	hashAlgo := sha256.New()
	hashAlgo.Write([]byte(input1))
	hashAlgo.Write([]byte(input2))
	dig := hashAlgo.Sum(nil)
	for i := 0; i < sha256.Size; i++ {
		digest[i] = dig[i]
	}
	return nil
}
