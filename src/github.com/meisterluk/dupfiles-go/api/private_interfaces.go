package api

// HashingAlgorithm is the generic interface for all hash algorithm
// implementations to be used
type HashingAlgorithm interface {
	// HashFile content or any data of a file according to some hashFileSpec.
	// The file is specified with a relative path and the hash stored in digest.
	HashFile(spec HashingSpec, relPath string, digest []byte) error

	// HashString hashes a given string and stores the result in digest
	HashString(hashme string, digest []byte) error

	// HashTwoHashes hashes two given hashes
	HashTwoHashes(input1 []byte, input2 []byte, digest []byte) error

	// HashDirectory updates the parent hash value with the child hash value
	HashDirectory(parent []byte, child []byte) error

	// String returns the hash algorithm's name in human-readable form
	String() string
}
