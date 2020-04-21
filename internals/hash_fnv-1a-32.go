package internals

import (
	"hash"
	"hash/fnv"
	"io"
	"os"
)

// FNV1a_32 implements the Fowler–Noll–Vo non-cryptographic hash function
// invented by Glenn Fowler, Landon Curt Noll, and Kiem-Phong Vo (1991)
type FNV1a_32 struct {
	h hash.Hash
}

// NewFNV1a_32 defines returns a properly initialized FNV1a_32 instance
func NewFNV1a_32() *FNV1a_32 {
	c := new(FNV1a_32)
	c.h = fnv.New32a()
	return c
}

// Hash returns the hash state in a Hash instance
func (c *FNV1a_32) Hash() Hash {
	var hash [4]byte
	copy(hash[:], c.h.Sum([]byte{}))
	return Hash32Bits(hash)
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *FNV1a_32) Name() string {
	return "fnv-1a-32"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *FNV1a_32) NewCopy() HashAlgorithm {
	return NewFNV1a_32()
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *FNV1a_32) ReadFile(filepath string) error {
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
func (c *FNV1a_32) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
