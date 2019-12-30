package internals

import (
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// SHA512 implements the Merkle–Damgård structure based, cryptographic hash algorithm invented by NSA (2001)
type SHA512 struct {
	h   hash.Hash
	sum []byte
}

// NewSHA512 defines returns a properly initialized SHA512 instance
func NewSHA512() *SHA512 {
	c := new(SHA512)
	c.h = sha512.New()
	return c
}

// Size returns the number of bytes of the hashsum
func (c *SHA512) Size() int {
	return c.h.Size()
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

	c.sum = c.h.Sum([]byte{})
	return nil
}

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *SHA512) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *SHA512) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *SHA512) Digest() []byte {
	return c.sum
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *SHA512) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

// HashAlgorithm returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHA512) HashAlgorithm() string {
	return "sha-512"
}
