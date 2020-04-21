package internals

import (
	"crypto/md5"
	"hash"
	"io"
	"os"
)

// MD5 implements the message-digest algorithm invented by Ronald Rivest (1991)
type MD5 struct {
	h hash.Hash
}

// NewMD5 defines returns a properly initialized MD5 instance
func NewMD5() *MD5 {
	c := new(MD5)
	c.h = md5.New()
	return c
}

// Hash returns the hash state in a Hash instance
func (c *MD5) Hash() Hash {
	var hash [16]byte
	data := c.h.Sum([]byte{})
	copy(hash[:], data)
	return Hash128Bits(hash)
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *MD5) Name() string {
	return "md5"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *MD5) NewCopy() HashAlgorithm {
	return NewMD5()
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

	return nil
}

// ReadBytes provides an interface to update the hash state with individual bytes
func (c *MD5) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
