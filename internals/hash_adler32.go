package internals

import (
	"encoding/hex"
	"hash"
	"hash/adler32"
	"io"
	"os"
)

type Adler32 struct {
	h   hash.Hash32
	sum uint32
}

func NewAdler32() *Adler32 {
	c := new(Adler32)
	c.h = adler32.New()
	return c
}

func (c *Adler32) Size() int {
	return c.h.Size()
}

func (c *Adler32) ReadFile(filepath string) error {
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

	c.sum = c.h.Sum32()
	return nil
}

func (c *Adler32) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *Adler32) Reset() {
	c.h.Reset()
}

func (c *Adler32) Digest() []byte {
	return []byte{
		byte(c.sum >> 24),
		byte(c.sum >> 16),
		byte(c.sum >> 8),
		byte(c.sum >> 0),
	}
}

func (c *Adler32) HexDigest() string {
	return hex.EncodeToString(c.Digest())
}

func (c *Adler32) HashAlgorithm() string {
	return "adler32"
}
