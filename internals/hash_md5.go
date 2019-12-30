package internals

import (
	"crypto/md5"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// MD5 implements the message-digest algorithm invented by Ronald Rivest (1991)
type MD5 struct {
	h   hash.Hash
	sum []byte
}

// NewMD5 defines returns a properly initialized MD5 instance
func NewMD5() *MD5 {
	c := new(MD5)
	c.h = md5.New()
	return c
}

// Size returns the number of bytes of the hashsum
func (c *MD5) Size() int {
	return c.h.Size()
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *MD5) ReadFile(filepath string) error {
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
func (c *MD5) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}

// Reset resets the hash state to its initial state.
// After this call functions like `ReadFile` or `ReadBytes` can be called.
func (c *MD5) Reset() {
	c.h.Reset()
}

// Digest returns the digest resulting from the hash state
func (c *MD5) Digest() []byte {
	return c.sum
}

// HexDigest returns the hash state digest encoded in a hexadecimal string
func (c *MD5) HexDigest() string {
	return hex.EncodeToString(c.sum)
}

// HashAlgorithm returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *MD5) HashAlgorithm() string {
	return "md5"
}
