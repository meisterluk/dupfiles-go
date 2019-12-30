package internals

import (
	"encoding/hex"
	"io"
	"os"

	"golang.org/x/crypto/sha3"
)

// SHAKE256_64 implements the SHAKE hash algorithm with 64bit output and a security claim of 32 bits
type SHAKE256_64 struct {
	h   sha3.ShakeHash
	sum [64]byte
}

// NewSHAKE256_64 defines returns a properly initialized SHAKE256-64 instance
func NewSHAKE256_64() *SHAKE256_64 {
	c := new(SHAKE256_64)
	c.h = sha3.NewShake256()
	return c
}

// Size returns the number of bytes of the hashsum
func (c *SHAKE256_64) Size() int {
	return 64
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *SHAKE256_64) ReadFile(filepath string) error {
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

	_, err = c.h.Read(c.sum[:])
	return err
}

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *SHAKE256_64) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *SHAKE256_64) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *SHAKE256_64) Digest() []byte {
	return c.sum[:]
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *SHAKE256_64) HexDigest() string {
	return hex.EncodeToString(c.sum[:])
}

// HashAlgorithm returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHAKE256_64) HashAlgorithm() string {
	return "shake256-64"
}
