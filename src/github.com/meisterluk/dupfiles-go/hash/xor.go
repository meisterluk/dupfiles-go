package hash

// XORDirHash updates parent hash value with xor of child hash value
func XORDirHash(parent []byte, child []byte) error {
	for i := 0; i < len(parent); i++ {
		parent[i] ^= child[i]
	}
	return nil
}
