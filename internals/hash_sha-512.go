package internals

import (
	"crypto/sha512"
	"hash"
	"io"
	"os"
)

// SHA512 implements the Merkle–Damgård structure based, cryptographic hash algorithm invented by NSA (2001)
type SHA512 struct {
	h hash.Hash
}

// NewSHA512 defines returns a properly initialized SHA512 instance
func NewSHA512() *SHA512 {
	c := new(SHA512)
	c.h = sha512.New()
	return c
}

// Hash returns the hash state in a Hash instance
func (c *SHA512) Hash() Hash {
	hash := make(Hash, 64)
	copy(hash[:], c.h.Sum([]byte{}))
	return hash
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHA512) Name() string {
	return "sha-512"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *SHA512) NewCopy() HashAlgorithm {
	return NewSHA512()
}

// OutputSize returns the hash output size in bytes
func (c *SHA512) OutputSize() int {
	return 64
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *SHA512) ReadFile(filepath string) error {
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
func (c *SHA512) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
