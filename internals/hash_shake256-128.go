package internals

import (
	"encoding/hex"
	"io"
	"os"

	"golang.org/x/crypto/sha3"
)

// SHAKE256_128 implements the SHAKE hash algorithm with 128bit output and a security claim of 32 bits
type SHAKE256_128 struct {
	h   sha3.ShakeHash
	buf [128]byte
}

// NewSHAKE256_128 defines returns a properly initialized SHAKE256-128 instance
func NewSHAKE256_128() *SHAKE256_128 {
	c := new(SHAKE256_128)
	c.h = sha3.NewShake256()
	return c
}

// Size returns the number of bytes of the hashsum
func (c *SHAKE256_128) Size() int {
	return 128
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *SHAKE256_128) ReadFile(filepath string) error {
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

	_, err = c.h.Read(c.buf[:])
	return err
}

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *SHAKE256_128) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	if err != nil {
		return err
	}

	_, err = c.h.Read(c.buf[:])
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *SHAKE256_128) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *SHAKE256_128) Digest() []byte {
	return c.buf[:]
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *SHAKE256_128) HexDigest() string {
	return hex.EncodeToString(c.buf[:])
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHAKE256_128) Name() string {
	return "shake256-128"
}
