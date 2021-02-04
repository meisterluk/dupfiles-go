package internals

import (
	"crypto/sha1"
	"hash"
	"io"
	"os"
)

// SHA1 implements the former cryptographic hash function invented by the Capstone project (1993)
type SHA1 struct {
	h hash.Hash
}

// NewSHA1 defines returns a properly initialized SHA1 instance
func NewSHA1() *SHA1 {
	c := new(SHA1)
	c.h = sha1.New()
	return c
}

// Hash returns the hash state in a Hash instance
func (c *SHA1) Hash() Hash {
	hash := make(Hash, 20)
	copy(hash[:], c.h.Sum([]byte{}))
	return hash
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHA1) Name() string {
	return "sha-1"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *SHA1) NewCopy() HashAlgorithm {
	return NewSHA1()
}

// OutputSize returns the hash output size in bytes
func (c *SHA1) OutputSize() int {
	return 20
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *SHA1) ReadFile(filepath string) error {
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
func (c *SHA1) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
