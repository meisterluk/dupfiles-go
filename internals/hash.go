package internals

import (
	"fmt"
)

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
	HashAlgorithm() string
}

// HashForHashAlgo returns a Hash instance of a given type
// determined by the string argument.
func HashForHashAlgo(hashAlgo string) (Hash, error) {
	switch hashAlgo {
	case "crc64":
		return NewCRC64(), nil
	case "crc32":
		return NewCRC32(), nil
	case "fnv-1-32":
		return NewFNV1_32(), nil
	case "fnv-1-64":
		return NewFNV1_64(), nil
	case "fnv-1-128":
		return NewFNV1_128(), nil
	case "fnv-1a-32":
		return NewFNV1a_32(), nil
	case "fnv-1a-64":
		return NewFNV1a_64(), nil
	case "fnv-1a-128":
		return NewFNV1a_128(), nil
	case "adler32":
		return NewAdler32(), nil
	case "md5":
		return NewMD5(), nil
	case "sha-1":
		return NewSHA1(), nil
	case "sha-256":
		return NewSHA256(), nil
	case "sha-512":
		return NewSHA512(), nil
	case "sha-3":
		return NewSHA3_512(), nil
	case "shake256-64":
		return NewSHAKE256_128(), nil
	}
	return NewCRC64(), fmt.Errorf(`unknown hash algorithm '%s'`, hashAlgo)
}

// OutputSizeForHashAlgo returns the output size in bytes for a given hash algorithm.
func OutputSizeForHashAlgo(hashAlgo string) int {
	switch hashAlgo {
	case "crc64":
		return 8
	case "crc32":
		return 4
	case "fnv-1-32":
		return 4
	case "fnv-1-64":
		return 8
	case "fnv-1-128":
		return 16
	case "fnv-1a-32":
		return 4
	case "fnv-1a-64":
		return 8
	case "fnv-1a-128":
		return 16
	case "adler32":
		return 4
	case "md5":
		return 16
	case "sha-1":
		return 20
	case "sha-256":
		return 32
	case "sha-512":
		return 64
	case "sha-3":
		return 64
	case "shake256-64":
		return 8
	}
	return 0
}

// SupportedHashAlgorithms returns the list of supported hash algorithms.
// The slice contains specified hash algorithm identifiers
func SupportedHashAlgorithms() []string {
	return []string{
		"crc64", "crc32", "fnv-1-32", "fnv-1-64", "fnv-1-128",
		"fnv-1a-32", "fnv-1a-64", "fnv-1a-128", "adler32",
		"md5", "sha-1", "sha-256", "sha-512", "sha-3",
		"shake256-64",
	}
}
