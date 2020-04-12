package internals

import (
	"encoding/hex"
	"log"
)

// DigestData essentially represents a list of all digests occuring at least twice.
// It essentially maintains data, which stores a list of digests with metadata.
type DigestData struct {
	digestSize         int // equals the size of one entry in slice data[n]
	totalDigests       uint64
	totalUniqueDigests uint64
	data               [256][]byte
	// INVARIANT digests are unique within data ⇒ (first-byte, index) uniquely identifies a digest
}

// NewDigestData creates a new DigestData struct and initializes the contained data
func NewDigestData(digestSize int, itemsPerByte int) *DigestData {
	d := new(DigestData)
	d.digestSize = digestSize
	for i := 0; i < 256; i++ {
		d.data[i] = make([]byte, 0, itemsPerByte)
	}
	return d
}

// Add adds a digest to the DigestData set and returns its index
// as well as a boolean. The boolean is false
// iff the digest has not been found and was added explicitly
func (d *DigestData) Add(digest []byte) (int, bool) {
	d.totalDigests++

	digestSuffix := digest[1:]
	firstByte := digest[0]

	// REMINDER entries in data[n] consist of {
	//    "digest" of digest-size ‖
	//    "disabled" of one bit ‖
	//    "dups" of 7 bits
	// }
	for i := 0; i*d.digestSize < len(d.data[firstByte]); i++ {
		itemDigestSuffix := d.data[firstByte][i*d.digestSize : (i+1)*d.digestSize-1]
		if EqByteSlices(itemDigestSuffix, digestSuffix) {
			// set dups byte to "min(dups + 1, MaxCountInDataStructure)".
			dups := uint(d.data[firstByte][i*d.digestSize+d.digestSize-1])
			if dups != MaxCountInDataStructure {
				dups++
			}
			d.data[firstByte][(i+1)*d.digestSize-1] = byte(dups)
			return i, true
		}
	}

	d.data[firstByte] = append(d.data[firstByte], digestSuffix...)
	d.data[firstByte] = append(d.data[firstByte], 0)
	d.totalUniqueDigests++
	return len(d.data[firstByte])/d.digestSize - 1, false
}

// Digest returns the byte array for the digest identified by (firstByte, idx)
func (d *DigestData) Digest(firstByte byte, idx int) []byte {
	digest := make([]byte, d.digestSize)
	copy(digest[0:1], []byte{firstByte})
	copy(digest[1:], d.data[firstByte][idx*d.digestSize:(idx+1)*d.digestSize-1])
	return digest
}

// DigestSuffix returns all-but-first bytes of the byte array
// for the digest identified by (firstByte, idx)
func (d *DigestData) DigestSuffix(firstByte byte, idx int) []byte {
	return d.data[firstByte][idx*d.digestSize : (idx+1)*d.digestSize-1]
}

// Disable declares the digestidentified by (firstByte, idx) as "disabled"
func (d *DigestData) Disable(firstByte byte, idx int) {
	d.data[firstByte][(idx+1)*d.digestSize-1] = d.data[firstByte][(idx+1)*d.digestSize-1] | (MaxCountInDataStructure + 1)
}

// Disabled returns whether the disabled bit for a digest has been set
func (d *DigestData) Disabled(firstByte byte, idx int) bool {
	return int(d.data[firstByte][(idx+1)*d.digestSize-1])&(MaxCountInDataStructure+1) > 0
}

// Duplicates returns the number of duplicates for the digest identified by (firstByte, idx).
// Be aware that number MaxCountInDataStructure means "MaxCountInDataStructure or more".
func (d *DigestData) Duplicates(firstByte byte, idx int) int {
	return int(d.data[firstByte][(idx+1)*d.digestSize-1]) & MaxCountInDataStructure
}

// IndexOf returns the index (2nd part of entry identifier) of the given digest
// as well as a boolean indicating existence of this digest
func (d *DigestData) IndexOf(digest []byte) (int, bool) {
	digestList := d.data[digest[0]]
	for i := 0; i*d.digestSize < len(digestList); i++ {
		if EqByteSlices(digest[1:], digestList[i*d.digestSize:(i+1)*d.digestSize-1]) {
			return i, true
		}
	}
	return 0, false
}

// Dump dumps the data within the data structure for debugging purposes
func (d *DigestData) Dump() {
	log.Println("<data>")
	for i := 0; i < 256; i++ {
		firstByte := hex.EncodeToString([]byte{byte(i)})
		for j := 0; j*d.digestSize < len(d.data[i]); j++ {
			digestSuffix := d.data[i][j*d.digestSize : (j+1)*d.digestSize-1]
			disabled := (d.data[i][(j+1)*d.digestSize-1] & (MaxCountInDataStructure + 1)) > 0
			dups := int(d.data[i][(j+1)*d.digestSize-1] & MaxCountInDataStructure)
			log.Printf("%s%s‖%t‖%d ", firstByte, hex.EncodeToString(digestSuffix), disabled, dups)

			if !EqByteSlices(digestSuffix, d.DigestSuffix(byte(i), j)) {
				panic("digest do not match")
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
