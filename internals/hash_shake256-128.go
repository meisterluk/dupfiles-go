package internals

import (
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

// Hash returns the hash state in a Hash instance
func (c *SHAKE256_128) Hash() Hash {
	var hash [16]byte
	copy(hash[:], c.buf[:])
	return Hash128Bits(hash)
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *SHAKE256_128) Name() string {
	return "shake256-128"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *SHAKE256_128) NewCopy() HashAlgorithm {
	return NewSHAKE256_128()
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
