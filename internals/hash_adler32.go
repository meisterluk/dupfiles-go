package internals

import (
	"hash"
	"hash/adler32"
	"io"
	"os"
)

// Adler32 implements the checksum algorithm invented by Mark Adler (1995)
type Adler32 struct {
	h hash.Hash32
}

// NewAdler32 defines returns a properly initialized Adler32 instance
func NewAdler32() *Adler32 {
	c := new(Adler32)
	c.h = adler32.New()
	return c
}

// Hash returns the hash state in a Hash instance
func (c *Adler32) Hash() Hash {
	sum := c.h.Sum32()
	hash := make(Hash, 4)
	for i := 0; i < 4; i++ {
		hash[i] = byte(sum >> (24 - 8*i))
	}
	return hash
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *Adler32) Name() string {
	return "adler32"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *Adler32) NewCopy() HashAlgorithm {
	return NewAdler32()
}

// OutputSize returns the hash output size in bytes
func (c *Adler32) OutputSize() int {
	return 4
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *Adler32) ReadFile(filepath string) error {
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
func (c *Adler32) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
