package internals

import "encoding/hex"

// Hash32Bits represents any hash with 32 bits output size
type Hash32Bits [4]byte

// Digest returns the hexadecimal nibble representation of this hash
func (h Hash32Bits) Digest() string {
	return hex.EncodeToString(h[:])
}

// FromData fills the Hash instance with given data
func (h Hash32Bits) FromData(data []byte) {
	for i := 0; i < len(h); i++ {
		h[i] = data[i]
	}
}

// Size returns the digest size in bytes
func (h Hash32Bits) Size() int {
	return 4
}

// ToData returns the byte array which is the raw hash value
func (h Hash32Bits) ToData() []byte {
	return h[:]
}

// XOR updates this hash by xoring this hash with the other hash.
// NOTE the caller needs to ensure both hashes have the same size.
func (h Hash32Bits) XOR(other Hash) {
	operand := other.ToData()
	for i := 0; i < 4; i++ {
		h[i] = h[i] ^ operand[i]
	}
}

// Hash64Bits represents any hash with 64 bits output size
type Hash64Bits [8]byte

// Digest returns the hexadecimal nibble representation of this hash
func (h Hash64Bits) Digest() string {
	return hex.EncodeToString(h[:])
}

// FromData fills the Hash instance with given data
func (h Hash64Bits) FromData(data []byte) {
	for i := 0; i < len(h); i++ {
		h[i] = data[i]
	}
}

// Size returns the digest size in bytes
func (h Hash64Bits) Size() int {
	return 8
}

// ToData returns the byte array which is the raw hash value
func (h Hash64Bits) ToData() []byte {
	return h[:]
}

// XOR updates this hash by xoring this hash with the other hash.
// NOTE the caller needs to ensure both hashes have the same size.
func (h Hash64Bits) XOR(other Hash) {
	operand := other.ToData()
	for i := 0; i < 8; i++ {
		h[i] = h[i] ^ operand[i]
	}
}

// Hash128Bits represents any hash with 128 bits output size
type Hash128Bits [16]byte

// Digest returns the hexadecimal nibble representation of this hash
func (h Hash128Bits) Digest() string {
	return hex.EncodeToString(h[:])
}

// FromData fills the Hash instance with given data
func (h Hash128Bits) FromData(data []byte) {
	for i := 0; i < len(h); i++ {
		h[i] = data[i]
	}
}

// Size returns the digest size in bytes
func (h Hash128Bits) Size() int {
	return 16
}

// ToData returns the byte array which is the raw hash value
func (h Hash128Bits) ToData() []byte {
	return h[:]
}

// XOR updates this hash by xoring this hash with the other hash.
// NOTE the caller needs to ensure both hashes have the same size.
func (h Hash128Bits) XOR(other Hash) {
	operand := other.ToData()
	for i := 0; i < 16; i++ {
		h[i] = h[i] ^ operand[i]
	}
}

// Hash160Bits represents any hash with 160 bits output size
type Hash160Bits [20]byte

// Digest returns the hexadecimal nibble representation of this hash
func (h Hash160Bits) Digest() string {
	return hex.EncodeToString(h[:])
}

// FromData fills the Hash instance with given data
func (h Hash160Bits) FromData(data []byte) {
	for i := 0; i < len(h); i++ {
		h[i] = data[i]
	}
}

// Size returns the digest size in bytes
func (h Hash160Bits) Size() int {
	return 20
}

// ToData returns the byte array which is the raw hash value
func (h Hash160Bits) ToData() []byte {
	return h[:]
}

// XOR updates this hash by xoring this hash with the other hash.
// NOTE the caller needs to ensure both hashes have the same size.
func (h Hash160Bits) XOR(other Hash) {
	operand := other.ToData()
	for i := 0; i < 20; i++ {
		h[i] = h[i] ^ operand[i]
	}
}

// Hash256Bits represents any hash with 256 bits output size
type Hash256Bits [32]byte

// Digest returns the hexadecimal nibble representation of this hash
func (h Hash256Bits) Digest() string {
	return hex.EncodeToString(h[:])
}

// FromData fills the Hash instance with given data
func (h Hash256Bits) FromData(data []byte) {
	for i := 0; i < len(h); i++ {
		h[i] = data[i]
	}
}

// Size returns the digest size in bytes
func (h Hash256Bits) Size() int {
	return 32
}

// ToData returns the byte array which is the raw hash value
func (h Hash256Bits) ToData() []byte {
	return h[:]
}

// XOR updates this hash by xoring this hash with the other hash.
// NOTE the caller needs to ensure both hashes have the same size.
func (h Hash256Bits) XOR(other Hash) {
	operand := other.ToData()
	for i := 0; i < 32; i++ {
		h[i] = h[i] ^ operand[i]
	}
}

// Hash512Bits represents any hash with 512 bits output size
type Hash512Bits [64]byte

// Digest returns the hexadecimal nibble representation of this hash
func (h Hash512Bits) Digest() string {
	return hex.EncodeToString(h[:])
}

// FromData fills the Hash instance with given data
func (h Hash512Bits) FromData(data []byte) {
	for i := 0; i < len(h); i++ {
		h[i] = data[i]
	}
}

// Size returns the digest size in bytes
func (h Hash512Bits) Size() int {
	return 64
}

// ToData returns the byte array which is the raw hash value
func (h Hash512Bits) ToData() []byte {
	return h[:]
}

// XOR updates this hash by xoring this hash with the other hash.
// NOTE the caller needs to ensure both hashes have the same size.
func (h Hash512Bits) XOR(other Hash) {
	operand := other.ToData()
	for i := 0; i < 64; i++ {
		h[i] = h[i] ^ operand[i]
	}
}
