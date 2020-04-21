package internals

import (
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

// Hash returns the hash state in a Hash instance
func (c *FNV1_64) Hash() Hash {
	hash := make(Hash, 8)
	copy(hash[:], c.h.Sum([]byte{}))
	return hash
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *FNV1_64) Name() string {
	return "fnv-1-64"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *FNV1_64) NewCopy() HashAlgorithm {
	return NewFNV1_64()
}

// OutputSize returns the hash output size in bytes
func (c *FNV1_64) OutputSize() int {
	return 8
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
