package internals

import (
	"hash"
	"hash/crc64"
	"io"
	"os"
)

// CRC64 implements the cyclic redundancy check invented by W. Wesley Peterson (1961)
type CRC64 struct {
	h hash.Hash64
}

// NewCRC64 defines returns a properly initialized CRC64 instance
func NewCRC64() *CRC64 {
	crc64Table := crc64.MakeTable(crc64.ISO)

	c := new(CRC64)
	c.h = crc64.New(crc64Table)

	return c
}

// Hash returns the hash state in a Hash instance
func (c *CRC64) Hash() Hash {
	sum := c.h.Sum64()
	return Hash64Bits([8]byte{
		byte(sum >> 56),
		byte(sum >> 48),
		byte(sum >> 40),
		byte(sum >> 32),
		byte(sum >> 24),
		byte(sum >> 16),
		byte(sum >> 8),
		byte(sum >> 0),
	})
}

// Name returns the hash algorithm's name
// in accordance with the dupfiles design document
func (c *CRC64) Name() string {
	return "crc64"
}

// NewCopy returns a copy of this hash algorithm with freshly initialized hash state
func (c *CRC64) NewCopy() HashAlgorithm {
	return NewCRC64()
}

// ReadFile provides an interface to update the hash state with the content of an entire file
func (c *CRC64) ReadFile(filepath string) error {
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
func (c *CRC64) ReadBytes(data []byte) error {
	_, err := c.h.Write(data)
	return err
}
