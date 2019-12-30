package internals

import (
	"encoding/hex"
	"hash"
	"hash/crc32"
	"io"
	"os"
)

// CRC32 implements the cyclic redundancy check invented by W. Wesley Peterson (1961)
type CRC32 struct {
	h   hash.Hash32
	sum uint32
}

// NewCRC32 defines returns a properly initialized CRC32 instance
func NewCRC32() *CRC32 {
	crc32Table := crc32.MakeTable(crc32.IEEE)

	c := new(CRC32)
	c.h = crc32.New(crc32Table)

	return c
}

// Size returns the number of bytes of the hashsum
func (c *CRC32) Size() int {
	return c.h.Size()
}

// ReadFile provides an interface to update the hash state with the content of an entire file
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

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *CRC32) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *CRC32) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *CRC32) Digest() []byte {
	return []byte{
		byte(c.sum >> 24),
		byte(c.sum >> 16),
		byte(c.sum >> 8),
		byte(c.sum >> 0),
	}
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *CRC32) HexDigest() string {
	return hex.EncodeToString(c.Digest())
}

// HashAlgorithm returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *CRC32) HashAlgorithm() string {
	return "crc32"
}
