package internals

import (
	"encoding/hex"
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

// Size returns the number of bytes of the hashsum
func (c *SHA3_512) Size() int {
	return c.h.Size()
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

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *SHA3_512) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *SHA3_512) Digest() []byte {
	return c.h.Sum([]byte{})
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *SHA3_512) HexDigest() string {
	return hex.EncodeToString(c.Digest())
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHA3_512) Name() string {
	return "sha-3-512"
}
