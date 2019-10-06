package internals

import (
	"crypto/md5"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

type MD5 struct {
	h   hash.Hash
	sum []byte
}

func NewMD5() *MD5 {
	c := new(MD5)
	c.h = md5.New()
	return c
}

func (c *MD5) Size() int {
	return c.h.Size()
}

func (c *MD5) ReadFile(filepath string) error {
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

func (c *MD5) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *MD5) Reset() {
	c.h.Reset()
}

func (c *MD5) Digest() []byte {
	return c.sum
}

func (c *MD5) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

func (c *MD5) HashAlgorithm() string {
	return "md5"
}
