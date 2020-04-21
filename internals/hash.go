package internals

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// HashAlgorithm is a custom interface to define operations
// a hash algorithm needs to support to include it in dupfiles.
// HashAlgorithm is implemented by every hash algorithm specified
// in the dupfiles design document (see adjacent “hash_*.go” files).
type HashAlgorithm interface {
	// get state hash
	Hash() Hash
	// get string representation of this hash algorithm
	Name() string
	// return a copy of this hash algorithm with freshly initialized hash state
	NewCopy() HashAlgorithm
	// returns the output size in bytes
	OutputSize() int
	// update hash state with file content at given filepath
	ReadFile(string) error
	// update hash state with given bytes
	ReadBytes([]byte) error
}

// HashAlgo is an alias for uint16. Specifically it is an index
// into the table of all registered hash algorithms.
type HashAlgo uint16

// HashAlgos contains a complete list of all hash algorithms
type HashAlgos struct{}

// Hash represents some hash value of a hash algorithm
type Hash []byte

// --- Abstractions finished. Now we consider the actual implementations ---

const (
	// HashCRC64 → Cyclic redundancy check, 64 bits output
	HashCRC64 HashAlgo = iota
	// HashCRC32 → Cyclic redundancy check, 32 bits output
	HashCRC32 HashAlgo = iota
	// HashFNV1_32 → Fowler–Noll–Vo hash function, 32 bits output
	HashFNV1_32 HashAlgo = iota
	// HashFNV1_64 → Fowler–Noll–Vo hash function, 64 bits output
	HashFNV1_64 HashAlgo = iota
	// HashFNV1_128 → Fowler–Noll–Vo hash function, 128 bits output
	HashFNV1_128 HashAlgo = iota
	// HashFNV1A32 → Fowler–Noll–Vo 1a hash function, 32 bits output
	HashFNV1A32 HashAlgo = iota
	// HashFNV1A64 → Fowler–Noll–Vo 1a hash function, 64 bits output
	HashFNV1A64 HashAlgo = iota
	// HashFNV1A128 → Fowler–Noll–Vo 1a hash function, 128 bits output
	HashFNV1A128 HashAlgo = iota
	// HashADLER32 → Mark Adler's checksum algorithm, 32 bits output
	HashADLER32 HashAlgo = iota
	// HashMD5 → Message-digest algorithm, 128 bits output
	HashMD5 HashAlgo = iota
	// HashSHA1 → hash function, 160 bits output
	HashSHA1 HashAlgo = iota
	// HashSHA256 → cryptographic hash function, 256 bits output
	HashSHA256 HashAlgo = iota
	// HashSHA512 → cryptographic hash function, 512 bits output
	HashSHA512 HashAlgo = iota
	// HashSHA3_512 → cryptographic hash function, 512 bits output
	HashSHA3_512 HashAlgo = iota
	// HashSHAKE256_64 → cryptographic hash function, 128 bits output
	HashSHAKE256_64 HashAlgo = iota
)

// CountHashAlgos returns the total number of registered hash algorithms
const CountHashAlgos = 15

// Algorithm returns a HashAlgorithm instance for the given hash algorithm name
func (h HashAlgo) Algorithm() HashAlgorithm {
	switch h {
	case HashCRC64:
		return NewCRC64()
	case HashCRC32:
		return NewCRC32()
	case HashFNV1_32:
		return NewFNV1_32()
	case HashFNV1_64:
		return NewFNV1_64()
	case HashFNV1_128:
		return NewFNV1_128()
	case HashFNV1A32:
		return NewFNV1a_32()
	case HashFNV1A64:
		return NewFNV1a_64()
	case HashFNV1A128:
		return NewFNV1a_128()
	case HashADLER32:
		return NewAdler32()
	case HashMD5:
		return NewMD5()
	case HashSHA1:
		return NewSHA1()
	case HashSHA256:
		return NewSHA256()
	case HashSHA512:
		return NewSHA512()
	case HashSHA3_512:
		return NewSHA3_512()
	case HashSHAKE256_64:
		return NewSHAKE256_128()
	}
	return HashAlgos{}.Default().Algorithm()
}

// Default returns the default hash algorithm
func (h HashAlgos) Default() HashAlgo {
	return HashFNV1A128
}

// FromString returns a HashAlgo instance matching the hash algorithm's name as a string
func (h HashAlgos) FromString(name string) (HashAlgo, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	for i := 0; i < CountHashAlgos; i++ {
		h := HashAlgo(i)
		if h.Algorithm().Name() == name {
			return h, nil
		}
	}
	return h.Default(), fmt.Errorf(`expected hash algorithm name; got unknown name '%q'`, name)
}

// Names returns the list of names of supported hash algorithms.
func (h HashAlgos) Names() []string {
	list := make([]string, CountHashAlgos)
	for i := 0; i < CountHashAlgos; i++ {
		list[i] = HashAlgo(i).Algorithm().Name()
	}
	return list
}

// Digest returns the hexadecimal nibble representation of a hash value (also called “digest”)
func (h Hash) Digest() string {
	return hex.EncodeToString(h)
}

// XOR updates this hash value by xoring it with the other hash value.
// NOTE the caller needs to ensure both hashes have the same size.
func (h Hash) XOR(other Hash) {
	for i := 0; i < len(h); i++ {
		h[i] ^= other[i]
	}
}
