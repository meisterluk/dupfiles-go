package internals

import (
	"fmt"
	"strings"
)

type hashAlgo string

const (
	hashCRC64       hashAlgo = `crc64`
	hashCRC32       hashAlgo = `crc32`
	hashFNV1_32     hashAlgo = `fnv-1-32`
	hashFNV1_64     hashAlgo = `fnv-1-64`
	hashFNV1_128    hashAlgo = `fnv-1-128`
	hashFNV1A32     hashAlgo = `fnv-1a-32`
	hashFNV1A64     hashAlgo = `fnv-1a-64`
	hashFNV1A128    hashAlgo = `fnv-1a-128`
	hashADLER32     hashAlgo = `adler32`
	hashMD5         hashAlgo = `md5`
	hashSHA1        hashAlgo = `sha-1`
	hashSHA256      hashAlgo = `sha-256`
	hashSHA512      hashAlgo = `sha-512`
	hashSHA3_512    hashAlgo = `sha-3-512`
	hashSHAKE256_64 hashAlgo = `shake256-64`
)

const defaultHash hashAlgo = hashCRC64

// SupportedHashAlgorithms returns the list of supported hash algorithms.
// The slice contains specified hash algorithm identifiers
func SupportedHashAlgorithms() []string {
	return []string{
		string(hashCRC64),
		string(hashCRC32),
		string(hashFNV1_32),
		string(hashFNV1_64),
		string(hashFNV1_128),
		string(hashFNV1A32),
		string(hashFNV1A64),
		string(hashFNV1A128),
		string(hashADLER32),
		string(hashMD5),
		string(hashSHA1),
		string(hashSHA256),
		string(hashSHA512),
		string(hashSHA3_512),
		string(hashSHAKE256_64),
	}
}

// OutputSize returns the output size in bytes for a given hash algorithm.
func (h hashAlgo) DigestSize() int {
	switch h {
	case hashCRC64:
		return 8
	case hashCRC32:
		return 4
	case hashFNV1_32:
		return 4
	case hashFNV1_64:
		return 8
	case hashFNV1_128:
		return 16
	case hashFNV1A32:
		return 4
	case hashFNV1A64:
		return 8
	case hashFNV1A128:
		return 16
	case hashADLER32:
		return 4
	case hashMD5:
		return 16
	case hashSHA1:
		return 20
	case hashSHA256:
		return 32
	case hashSHA512:
		return 64
	case hashSHA3_512:
		return 64
	case hashSHAKE256_64:
		return 8
	}
	return 0
}

// Algorithm returns a Hash instance for the given hash algorithm name.
func (h hashAlgo) Algorithm() Hash {
	switch h {
	case hashCRC64:
		return NewCRC64()
	case hashCRC32:
		return NewCRC32()
	case hashFNV1_32:
		return NewFNV1_32()
	case hashFNV1_64:
		return NewFNV1_64()
	case hashFNV1_128:
		return NewFNV1_128()
	case hashFNV1A32:
		return NewFNV1a_32()
	case hashFNV1A64:
		return NewFNV1a_64()
	case hashFNV1A128:
		return NewFNV1a_128()
	case hashADLER32:
		return NewAdler32()
	case hashMD5:
		return NewMD5()
	case hashSHA1:
		return NewSHA1()
	case hashSHA256:
		return NewSHA256()
	case hashSHA512:
		return NewSHA512()
	case hashSHA3_512:
		return NewSHA3_512()
	case hashSHAKE256_64:
		return NewSHAKE256_128()
	}
	return defaultHash.Algorithm()
}

// HashAlgorithmFromString returns a hashAlgo instance, give the hash algorithm's name as a string
func HashAlgorithmFromString(name string) (hashAlgo, error) {
	name = strings.ToLower(name)
	for _, algo := range SupportedHashAlgorithms() {
		if name == algo {
			return hashAlgo(algo), nil
		}
	}
	return defaultHash, fmt.Errorf(`unknown hash algorithm %q`, name)
}

// Hash is a custom interface to define operations
// a hash algorithm needs to support to include it in dupfiles
type Hash interface {
	// returns number of bytes of the digest
	Size() int
	// update hash state with data of file at given filepath
	ReadFile(string) error
	// update hash state with given bytes
	ReadBytes([]byte) error
	// reset hash state
	Reset()
	// get hash state digest
	Digest() []byte
	// get hash state digest represented as hexadecimal string
	HexDigest() string
	// get string representation of this hash algorithm
	Name() string
}
