package internals

import (
	"encoding/hex"
	"log"
)

// DigestData essentially represents a list of all digests occuring at least twice.
// It essentially maintains data, which stores a list of digests with metadata.
type DigestData struct {
	hashValueSize      int // equals the size of one entry in slice data[n]
	totalDigests       uint64
	totalUniqueDigests uint64
	data               [256][]byte
	// INVARIANT digests are unique within data ⇒ (first-byte, index) uniquely identifies a digest
}

// NewDigestData creates a new DigestData struct and initializes the contained data
func NewDigestData(hashValueSize int, itemsPerByte int) *DigestData {
	d := new(DigestData)
	d.hashValueSize = hashValueSize
	for i := 0; i < 256; i++ {
		d.data[i] = make([]byte, 0, itemsPerByte)
	}
	return d
}

// Add adds a hash value to the DigestData set and returns its index
// as well as a boolean. The boolean is false
// iff the hash value has not been found and was added explicitly
func (d *DigestData) Add(hashValue []byte) (int, bool) {
	d.totalDigests++

	hashValueSuffix := hashValue[1:]
	firstByte := hashValue[0]

	// REMINDER entries in data[n] consist of {
	//    "hash value" of hash-value-size ‖
	//    "disabled" of one bit ‖
	//    "dups" of 7 bits
	// }
	for i := 0; i*d.hashValueSize < len(d.data[firstByte]); i++ {
		itemHashValueSuffix := d.data[firstByte][i*d.hashValueSize : (i+1)*d.hashValueSize-1]
		if EqByteSlices(itemHashValueSuffix, hashValueSuffix) {
			// set dups byte to "min(dups + 1, MaxCountInDataStructure)".
			dups := uint(d.data[firstByte][i*d.hashValueSize+d.hashValueSize-1])
			if dups != MaxCountInDataStructure {
				dups++
			}
			d.data[firstByte][(i+1)*d.hashValueSize-1] = byte(dups)
			return i, true
		}
	}

	d.data[firstByte] = append(d.data[firstByte], hashValueSuffix...)
	d.data[firstByte] = append(d.data[firstByte], 0)
	d.totalUniqueDigests++
	return len(d.data[firstByte])/d.hashValueSize - 1, false
}

// Hash returns the byte array of the hash value identified by (firstByte, idx)
func (d *DigestData) Hash(firstByte byte, idx int) Hash {
	hashValue := make(Hash, d.hashValueSize)
	copy(hashValue[0:1], []byte{firstByte})
	copy(hashValue[1:], d.data[firstByte][idx*d.hashValueSize:(idx+1)*d.hashValueSize-1])
	return hashValue
}

// HashValueSuffix returns all-but-first bytes of the byte array
// for the hash value identified by (firstByte, idx)
func (d *DigestData) HashValueSuffix(firstByte byte, idx int) []byte {
	return d.data[firstByte][idx*d.hashValueSize : (idx+1)*d.hashValueSize-1]
}

// Disable declares the hash value identified by (firstByte, idx) as "disabled"
func (d *DigestData) Disable(firstByte byte, idx int) {
	d.data[firstByte][(idx+1)*d.hashValueSize-1] = d.data[firstByte][(idx+1)*d.hashValueSize-1] | (MaxCountInDataStructure + 1)
}

// Disabled returns whether the disabled bit for a hash value has been set
func (d *DigestData) Disabled(firstByte byte, idx int) bool {
	return int(d.data[firstByte][(idx+1)*d.hashValueSize-1])&(MaxCountInDataStructure+1) > 0
}

// Duplicates returns the number of duplicates for the hash value identified by (firstByte, idx).
// Be aware that number MaxCountInDataStructure means "MaxCountInDataStructure or more".
func (d *DigestData) Duplicates(firstByte byte, idx int) int {
	return int(d.data[firstByte][(idx+1)*d.hashValueSize-1]) & MaxCountInDataStructure
}

// IndexOf returns the index (2nd part of entry identifier) of the given hash value
// as well as a boolean indicating existence of this hash value
func (d *DigestData) IndexOf(hashValue []byte) (int, bool) {
	hashValueList := d.data[hashValue[0]]
	for i := 0; i*d.hashValueSize < len(hashValueList); i++ {
		if EqByteSlices(hashValue[1:], hashValueList[i*d.hashValueSize:(i+1)*d.hashValueSize-1]) {
			return i, true
		}
	}
	return 0, false
}

// Dump dumps the data within the data structure for debugging purposes
func (d *DigestData) Dump() {
	log.Println(`<data columns="hashvalue‖disabled‖dups">`)
	for i := 0; i < 256; i++ {
		firstByte := hex.EncodeToString([]byte{byte(i)})
		for j := 0; j*d.hashValueSize < len(d.data[i]); j++ {
			hashValueSuffix := d.data[i][j*d.hashValueSize : (j+1)*d.hashValueSize-1]
			disabled := (d.data[i][(j+1)*d.hashValueSize-1] & (MaxCountInDataStructure + 1)) > 0
			dups := int(d.data[i][(j+1)*d.hashValueSize-1] & MaxCountInDataStructure)
			log.Printf("%s%s‖%t‖%d ", firstByte, hex.EncodeToString(hashValueSuffix), disabled, dups)

			if !EqByteSlices(hashValueSuffix, d.HashValueSuffix(byte(i), j)) {
				panic("hash values do not match")
			}
			if disabled != d.Disabled(byte(i), j) {
				panic("disabled bits do not match")
			}
			if dups != d.Duplicates(byte(i), j) {
				panic("duplicates counts do not match")
			}
		}
	}
	log.Println("</data>")
}
