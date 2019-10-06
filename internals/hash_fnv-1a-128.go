package internals

import (
	"encoding/hex"
	"hash"
	"hash/fnv"
	"io"
	"os"
)

type FNV1a_128 struct {
	h   hash.Hash
	sum []byte
}

func NewFNV1a_128() *FNV1a_128 {
	c := new(FNV1a_128)
	c.h = fnv.New128a()
	return c
}

func (c *FNV1a_128) Size() int {
	return c.h.Size()
}

func (c *FNV1a_128) ReadFile(filepath string) error {
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

func (c *FNV1a_128) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *FNV1a_128) Reset() {
	c.h.Reset()
}

func (c *FNV1a_128) Digest() []byte {
	return c.sum
}

func (c *FNV1a_128) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

func (c *FNV1a_128) HashAlgorithm() string {
	return "fnv-1-128"
}
