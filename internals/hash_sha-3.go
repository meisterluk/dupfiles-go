package internals

import (
	"hash"
	"io"
	"os"

	"golang.org/x/crypto/sha3"
)

// SHA3_512 implements the sponge construction based hash algorithm
// invented by Guido Bertoni, Joan Daemen, Michaël Peeters, and Gilles Van Assche (2008)
type SHA3_512 struct {
	h hash.Hash
}

// NewSHA3_512 defines returns a properly initialized SHA3_512 instance
func NewSHA3_512() *SHA3_512 {
	c := new(SHA3_512)
	c.h = sha3.New512()
	return c
}

// Hash returns the hash state in a Hash instance
func (c *SHA3_512) Hash() Hash {
	hash := make(Hash, 64)
	copy(hash[:], c.h.Sum([]byte{}))
	return hash
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHA3_512) Name() string {
	return "sha-3-512"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *SHA3_512) NewCopy() HashAlgorithm {
	return NewSHA3_512()
}

// OutputSize returns the hash output size in bytes
func (c *SHA3_512) OutputSize() int {
	return 64
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *SHA3_512) ReadFile(filepath string) error {
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
func (c *SHA3_512) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
