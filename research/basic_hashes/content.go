package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"hash/crc64"
	"hash/fnv"
	"io"

	"golang.org/x/crypto/sha3"
)

func writeEmptyMode(w io.Writer) {
	w.Write([]byte(`dupfiles generates rεports
😊
`))
	/*fd, err := os.Open("example.txt")
		if err != nil {
			panic(err)
		}
		defer fd.Close()
		_, err = io.Copy(w, fd)
		if err != nil {
			panic(err)
		}

		for _, b := range []byte(`dupfiles generates rεports
	😊
	`) {
			fmt.Printf("%x ", b)
		}
		fmt.Println("")*/
}

func writeBasenameMode(w io.Writer) {
	w.Write([]byte("example.txt"))
	w.Write([]byte{0x1F})
	w.Write([]byte(`dupfiles generates rεports
😊
`))
}

func adler_32(emptyMode bool) {
	h := adler32.New()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	s := h.Sum32()
	dig := []byte{
		byte(s >> 24),
		byte(s >> 16),
		byte(s >> 8),
		byte(s >> 0),
	}
	fmt.Print("adler32: ")
	fmt.Println(hex.EncodeToString(dig))
}

func crc_32(emptyMode bool) {
	crc32Table := crc32.MakeTable(crc32.IEEE)
	h := crc32.New(crc32Table)
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	s := h.Sum32()
	dig := []byte{
		byte(s >> 24),
		byte(s >> 16),
		byte(s >> 8),
		byte(s >> 0),
	}
	fmt.Print("crc32: ")
	fmt.Println(hex.EncodeToString(dig))
}

func crc_64(emptyMode bool) {
	crc64Table := crc64.MakeTable(crc64.ISO)
	h := crc64.New(crc64Table)
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	s := h.Sum64()
	dig := []byte{
		byte(s >> 56),
		byte(s >> 48),
		byte(s >> 40),
		byte(s >> 32),
		byte(s >> 24),
		byte(s >> 16),
		byte(s >> 8),
		byte(s >> 0),
	}
	fmt.Print("crc64: ")
	fmt.Println(hex.EncodeToString(dig))
}

func fnv1_32(emptyMode bool) {
	h := fnv.New32()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("FNV1-32: ")
	fmt.Println(hex.EncodeToString(dig))
}

func fnv1_64(emptyMode bool) {
	h := fnv.New64()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("FNV1-64: ")
	fmt.Println(hex.EncodeToString(dig))
}

func fnv1_128(emptyMode bool) {
	h := fnv.New128()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("FNV1-128: ")
	fmt.Println(hex.EncodeToString(dig))
}

func fnv1a_32(emptyMode bool) {
	h := fnv.New32a()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("FNV1a-32: ")
	fmt.Println(hex.EncodeToString(dig))
}

func fnv1a_64(emptyMode bool) {
	h := fnv.New64a()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("FNV1a-64: ")
	fmt.Println(hex.EncodeToString(dig))
}

func fnv1a_128(emptyMode bool) {
	h := fnv.New128a()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("FNV1a-128: ")
	fmt.Println(hex.EncodeToString(dig))
}

func md_5(emptyMode bool) {
	h := md5.New()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("MD5: ")
	fmt.Println(hex.EncodeToString(dig))
}

func sha_1(emptyMode bool) {
	h := sha1.New()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("sha1: ")
	fmt.Println(hex.EncodeToString(dig))
}

func sha_256(emptyMode bool) {
	h := sha256.New()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("sha256: ")
	fmt.Println(hex.EncodeToString(dig))
}

func sha_512(emptyMode bool) {
	h := sha512.New()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("sha512: ")
	fmt.Println(hex.EncodeToString(dig))
}

func sha_3(emptyMode bool) {
	h := sha3.New512()
	if emptyMode {
		writeEmptyMode(h)
	} else {
		writeBasenameMode(h)
	}
	dig := h.Sum([]byte{})
	fmt.Print("sha3-512: ")
	fmt.Println(hex.EncodeToString(dig))
}

func main() {
	fmt.Println("== Basename mode ==")
	adler_32(false)
	crc_32(false)
	crc_64(false)
	fnv1_32(false)
	fnv1_64(false)
	fnv1_128(false)
	fnv1a_32(false)
	fnv1a_64(false)
	fnv1a_128(false)
	md_5(false)
	sha_1(false)
	sha_256(false)
	sha_512(false)
	sha_3(false)
	fmt.Println("== Empty mode ==")
	adler_32(true)
	crc_32(true)
	crc_64(true)
	fnv1_32(true)
	fnv1_64(true)
	fnv1_128(true)
	fnv1a_32(true)
	fnv1a_64(true)
	fnv1a_128(true)
	md_5(true)
	sha_1(true)
	sha_256(true)
	sha_512(true)
	sha_3(true)
}
