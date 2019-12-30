package internals

import (
	"encoding/hex"
	"hash"
	"hash/crc64"
	"io"
	"os"
)

// CRC64 implements the cyclic redundancy check invented by W. Wesley Peterson (1961)
type CRC64 struct {
	h   hash.Hash64
	sum uint64
}

// NewCRC64 defines returns a properly initialized CRC64 instance
func NewCRC64() *CRC64 {
	crc64Table := crc64.MakeTable(crc64.ISO)

	c := new(CRC64)
	c.h = crc64.New(crc64Table)

	return c
}

// Size returns the number of bytes of the hashsum
func (c *CRC64) Size() int {
	return c.h.Size()
}

// ReadFile provides an interface to update the hash state with the content of an entire file
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

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *CRC64) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *CRC64) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
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

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *CRC64) HexDigest() string {
	return hex.EncodeToString(c.Digest())
}

// HashAlgorithm returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *CRC64) HashAlgorithm() string {
	return "crc64"
}
