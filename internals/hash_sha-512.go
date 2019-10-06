package internals

import (
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

type SHA512 struct {
	h   hash.Hash
	sum []byte
}

func NewSHA512() *SHA512 {
	c := new(SHA512)
	c.h = sha512.New()
	return c
}

func (c *SHA512) Size() int {
	return c.h.Size()
}

func (c *SHA512) ReadFile(filepath string) error {
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

func (c *SHA512) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *SHA512) Reset() {
	c.h.Reset()
}

func (c *SHA512) Digest() []byte {
	return c.sum
}

func (c *SHA512) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

func (c *SHA512) HashAlgorithm() string {
	return "sha-512"
}
