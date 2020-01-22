package internals

import (
	"encoding/hex"
	"hash"
	"hash/fnv"
	"io"
	"os"
)

// FNV1_64 implements the Fowler–Noll–Vo non-cryptographic hash function
// invented by Glenn Fowler, Landon Curt Noll, and Kiem-Phong Vo (1991)
type FNV1_64 struct {
	h hash.Hash
}

// NewFNV1_64 defines returns a properly initialized FNV1_64 instance
func NewFNV1_64() *FNV1_64 {
	c := new(FNV1_64)
	c.h = fnv.New64()
	return c
}

// Size returns the number of bytes of the hashsum
func (c *FNV1_64) Size() int {
	return c.h.Size()
}

// ReadFile provides an interface to update the hash state with the content of an entire file
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

	return nil
}

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *FNV1_64) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *FNV1_64) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *FNV1_64) Digest() []byte {
	return c.h.Sum([]byte{})
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *FNV1_64) HexDigest() string {
	return hex.EncodeToString(c.Digest())
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *FNV1_64) Name() string {
	return "fnv-1-64"
}
