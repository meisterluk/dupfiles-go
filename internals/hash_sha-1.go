package internals

import (
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// SHA1 implements the former cryptographic hash function invented by the Capstone project (1993)
type SHA1 struct {
	h   hash.Hash
	sum []byte
}

// NewSHA1 defines returns a properly initialized SHA1 instance
func NewSHA1() *SHA1 {
	c := new(SHA1)
	c.h = sha1.New()
	return c
}

// Size returns the number of bytes of the hashsum
func (c *SHA1) Size() int {
	return c.h.Size()
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

	c.sum = c.h.Sum([]byte{})
	return nil
}

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *SHA1) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *SHA1) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *SHA1) Digest() []byte {
	return c.sum
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *SHA1) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

// HashAlgorithm returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHA1) HashAlgorithm() string {
	return "sha-1"
}
