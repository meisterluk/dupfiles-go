package internals

import (
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

type SHA1 struct {
	h   hash.Hash
	sum []byte
}

func NewSHA1() *SHA1 {
	c := new(SHA1)
	c.h = sha1.New()
	return c
}

func (c *SHA1) Size() int {
	return c.h.Size()
}

func (c *SHA1) ReadFile(filepath string) error {
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

func (c *SHA1) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *SHA1) Reset() {
	c.h.Reset()
}

func (c *SHA1) Digest() []byte {
	return c.sum
}

func (c *SHA1) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

func (c *SHA1) HashAlgorithm() string {
	return "sha-1"
}
