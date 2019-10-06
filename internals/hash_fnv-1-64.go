package internals

import (
	"encoding/hex"
	"hash"
	"hash/fnv"
	"io"
	"os"
)

type FNV1_64 struct {
	h   hash.Hash
	sum []byte
}

func NewFNV1_64() *FNV1_64 {
	c := new(FNV1_64)
	c.h = fnv.New64()
	return c
}

func (c *FNV1_64) Size() int {
	return c.h.Size()
}

func (c *FNV1_64) ReadFile(filepath string) error {
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

func (c *FNV1_64) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *FNV1_64) Reset() {
	c.h.Reset()
}

func (c *FNV1_64) Digest() []byte {
	return c.sum
}

func (c *FNV1_64) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

func (c *FNV1_64) HashAlgorithm() string {
	return "fnv-1-64"
}
