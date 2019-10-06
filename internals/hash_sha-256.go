package internals

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

type SHA256 struct {
	h   hash.Hash
	sum []byte
}

func NewSHA256() *SHA256 {
	c := new(SHA256)
	c.h = sha256.New()
	return c
}

func (c *SHA256) Size() int {
	return c.h.Size()
}

func (c *SHA256) ReadFile(filepath string) error {
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

func (c *SHA256) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *SHA256) Reset() {
	c.h.Reset()
}

func (c *SHA256) Digest() []byte {
	return c.sum
}

func (c *SHA256) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

func (c *SHA256) HashAlgorithm() string {
	return "sha-256"
}
