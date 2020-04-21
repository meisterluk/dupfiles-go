package internals

import (
	"crypto/sha256"
	"hash"
	"io"
	"os"
)

// SHA256 implements the Merkle–Damgård structure based, cryptographic hash algorithm invented by NSA (2001)
type SHA256 struct {
	h hash.Hash
}

// NewSHA256 defines returns a properly initialized SHA256 instance
func NewSHA256() *SHA256 {
	c := new(SHA256)
	c.h = sha256.New()
	return c
}

// Hash returns the hash state in a Hash instance
func (c *SHA256) Hash() Hash {
	hash := make(Hash, 32)
	copy(hash[:], c.h.Sum([]byte{}))
	return hash
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHA256) Name() string {
	return "sha-256"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *SHA256) NewCopy() HashAlgorithm {
	return NewSHA256()
}

// OutputSize returns the hash output size in bytes
func (c *SHA256) OutputSize() int {
	return 32
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *SHA256) ReadFile(filepath string) error {
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
func (c *SHA256) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
