package internals

import (
	"fmt"
	"strings"
)

// HashAlgo is an alias for string, but specifically can only
// be one of the identifiers for hash algorithms.
type HashAlgo string

const (
	HashCRC64       HashAlgo = `crc64`
	HashCRC32       HashAlgo = `crc32`
	HashFNV1_32     HashAlgo = `fnv-1-32`
	HashFNV1_64     HashAlgo = `fnv-1-64`
	HashFNV1_128    HashAlgo = `fnv-1-128`
	HashFNV1A32     HashAlgo = `fnv-1a-32`
	HashFNV1A64     HashAlgo = `fnv-1a-64`
	HashFNV1A128    HashAlgo = `fnv-1a-128`
	HashADLER32     HashAlgo = `adler32`
	HashMD5         HashAlgo = `md5`
	HashSHA1        HashAlgo = `sha-1`
	HashSHA256      HashAlgo = `sha-256`
	HashSHA512      HashAlgo = `sha-512`
	HashSHA3_512    HashAlgo = `sha-3-512`
	HashSHAKE256_64 HashAlgo = `shake256-64`
)

const defaultHash HashAlgo = HashCRC64

// SupportedHashAlgorithms returns the list of supported hash algorithms.
// The slice contains specified hash algorithm identifiers
func SupportedHashAlgorithms() []string {
	return []string{
		string(HashCRC64),
		string(HashCRC32),
		string(HashFNV1_32),
		string(HashFNV1_64),
		string(HashFNV1_128),
		string(HashFNV1A32),
		string(HashFNV1A64),
		string(HashFNV1A128),
		string(HashADLER32),
		string(HashMD5),
		string(HashSHA1),
		string(HashSHA256),
		string(HashSHA512),
		string(HashSHA3_512),
		string(HashSHAKE256_64),
	}
}

func isValidHashAlgo(Hashalgo string) bool {
	whitelist := []string{
		"crc64", "crc32", "fnv-1-32", "fnv-1-64", "fnv-1-128", "fnv-1a-32", "fnv-1a-64",
		"fnv-1a-128", "adler32", "md5", "sha-1", "sha-256", "sha-512", "sha-3",
		"shake256-128",
	}
	for _, item := range whitelist {
		if item == Hashalgo {
			return true
		}
	}

	return false
}

// DigestSize returns the output size in bytes for a given hash algorithm.
func (h HashAlgo) DigestSize() int {
	switch h {
	case HashCRC64:
		return 8
	case HashCRC32:
		return 4
	case HashFNV1_32:
		return 4
	case HashFNV1_64:
		return 8
	case HashFNV1_128:
		return 16
	case HashFNV1A32:
		return 4
	case HashFNV1A64:
		return 8
	case HashFNV1A128:
		return 16
	case HashADLER32:
		return 4
	case HashMD5:
		return 16
	case HashSHA1:
		return 20
	case HashSHA256:
		return 32
	case HashSHA512:
		return 64
	case HashSHA3_512:
		return 64
	case HashSHAKE256_64:
		return 8
	}
	return 0
}

// Algorithm returns a Hash instance for the given hash algorithm name.
func (h HashAlgo) Algorithm() Hash {
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
	return defaultHash.Algorithm()
}

// HashAlgorithmFromString returns a HashAlgo instance, give the hash algorithm's name as a string
func HashAlgorithmFromString(name string) (HashAlgo, error) {
	name = strings.ToLower(name)
	for _, algo := range SupportedHashAlgorithms() {
		if name == algo {
			return HashAlgo(algo), nil
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
