package internals

import (
	"hash"
	"hash/crc32"
	"io"
	"os"
)

// CRC32 implements the cyclic redundancy check invented by W. Wesley Peterson (1961)
type CRC32 struct {
	h hash.Hash32
}

// NewCRC32 defines returns a properly initialized CRC32 instance
func NewCRC32() *CRC32 {
	crc32Table := crc32.MakeTable(crc32.IEEE)

	c := new(CRC32)
	c.h = crc32.New(crc32Table)

	return c
}

// Hash returns the hash state in a Hash instance
func (c *CRC32) Hash() Hash {
	sum := c.h.Sum32()
	hash := make(Hash, 4)
	for i := 0; i < 4; i++ {
		hash[i] = byte(sum >> (24 - 8*i))
	}
	return hash
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *CRC32) Name() string {
	return "crc32"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *CRC32) NewCopy() HashAlgorithm {
	return NewCRC32()
}

// OutputSize returns the hash output size in bytes
func (c *CRC32) OutputSize() int {
	return 4
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *CRC32) ReadFile(filepath string) error {
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
func (c *CRC32) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
