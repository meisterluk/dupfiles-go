package internals

import (
	"encoding/hex"
	"hash"
	"hash/crc64"
	"io"
	"os"
)

type CRC64 struct {
	h   hash.Hash64
	sum uint64
}

func NewCRC64() *CRC64 {
	crc64Table := crc64.MakeTable(crc64.ISO)

	c := new(CRC64)
	c.h = crc64.New(crc64Table)

	return c
}

func (c *CRC64) Size() int {
	return c.h.Size()
}

func (c *CRC64) ReadFile(filepath string) error {
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

	c.sum = c.h.Sum64()
	return nil
}

func (c *CRC64) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *CRC64) Reset() {
	c.h.Reset()
}

func (c *CRC64) Digest() []byte {
	return []byte{
		byte(c.sum >> 56),
		byte(c.sum >> 48),
		byte(c.sum >> 40),
		byte(c.sum >> 32),
		byte(c.sum >> 24),
		byte(c.sum >> 16),
		byte(c.sum >> 8),
		byte(c.sum >> 0),
	}
}

func (c *CRC64) HexDigest() string {
	return hex.EncodeToString(c.Digest())
}

func (c *CRC64) HashAlgorithm() string {
	return "crc64"
}
