package internals

import (
	"encoding/hex"
	"hash"
	"hash/crc32"
	"io"
	"os"
)

type CRC32 struct {
	h   hash.Hash32
	sum uint32
}

func NewCRC32() *CRC32 {
	crc32Table := crc32.MakeTable(crc32.IEEE)

	c := new(CRC32)
	c.h = crc32.New(crc32Table)

	return c
}

func (c *CRC32) Size() int {
	return c.h.Size()
}

func (c *CRC32) ReadFile(filepath string) error {
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

func (c *CRC32) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

func (c *CRC32) Reset() {
	c.h.Reset()
}

func (c *CRC32) Digest() []byte {
	return []byte{
		byte(c.sum >> 24),
		byte(c.sum >> 16),
		byte(c.sum >> 8),
		byte(c.sum >> 0),
	}
}

func (c *CRC32) HexDigest() string {
	return hex.EncodeToString(c.Digest())
}

func (c *CRC32) HashAlgorithm() string {
	return "crc32"
}
