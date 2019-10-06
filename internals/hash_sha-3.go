package internals

import (
	"encoding/hex"
	"hash"
	"io"
	"os"

	"golang.org/x/crypto/sha3"
)

type SHA3_512 struct {
	h   hash.Hash
	sum []byte
}

func NewSHA3_512() *SHA3_512 {
	c := new(SHA3_512)
	c.h = sha3.New512()
	return c
}

func (c *SHA3_512) Size() int {
	return c.h.Size()
}

func (c *SHA3_512) ReadFile(filepath string) error {
	// open/close file
	fd, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer fd.Close()

	// read file
	_, err = io.Copy(c.h, fd)
	if err != nil {
		return err
	}

	c.sum = c.h.Sum([]byte{})
	return nil
}

func (c *SHA3_512) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *SHA3_512) Reset() {
	c.h.Reset()
}

func (c *SHA3_512) Digest() []byte {
	return c.sum
}

func (c *SHA3_512) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

func (c *SHA3_512) HashAlgorithm() string {
	return "sha-3-512"
}
