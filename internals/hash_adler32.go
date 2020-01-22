package internals

import (
	"encoding/hex"
	"hash"
	"hash/adler32"
	"io"
	"os"
)

// Adler32 implements the checksum algorithm invented by Mark Adler (1995)
type Adler32 struct {
	h hash.Hash32
}

// NewAdler32 defines returns a properly initialized Adler32 instance
func NewAdler32() *Adler32 {
	c := new(Adler32)
	c.h = adler32.New()
	return c
}

// Size returns the number of bytes of the hashsum
func (c *Adler32) Size() int {
	return c.h.Size()
}

// ReadFile provides an interface to update the hash state with the content of an entire file
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

	return nil
}

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *Adler32) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *Adler32) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *Adler32) Digest() []byte {
	sum := c.h.Sum32()
	return []byte{
		byte(sum >> 24),
		byte(sum >> 16),
		byte(sum >> 8),
		byte(sum >> 0),
	}
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *Adler32) HexDigest() string {
	return hex.EncodeToString(c.Digest())
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *Adler32) Name() string {
	return "adler32"
}
