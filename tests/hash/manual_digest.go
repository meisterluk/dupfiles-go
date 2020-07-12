package tests

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
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/crypto/sha3"
)

func hashMe(hashContent []byte, hashAlgo string) ([]byte, error) {
	var hashValue []byte

	switch hashAlgo {
	case `crc64`:
		crc64Table := crc64.MakeTable(crc64.ISO)
		h := crc64.New(crc64Table)
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		sum := h.Sum64()
		hashValue = make([]byte, 8)
		for i := 0; i < 8; i++ {
			hashValue[i] = byte(sum >> (56 - 8*i))
		}
	case `crc32`:
		crc32Table := crc32.MakeTable(crc32.IEEE)
		h := crc32.New(crc32Table)
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		sum := h.Sum32()
		hashValue = make([]byte, 4)
		for i := 0; i < 4; i++ {
			hashValue[i] = byte(sum >> (24 - 8*i))
		}
	case `fnv-1-32`:
		h := fnv.New32()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `fnv-1-64`:
		h := fnv.New64()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `fnv-1-128`:
		h := fnv.New128()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `fnv-1a-32`:
		h := fnv.New32a()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `fnv-1a-64`:
		h := fnv.New64a()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `fnv-1a-128`:
		h := fnv.New128a()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `adler32`:
		h := adler32.New()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		sum := h.Sum32()
		hashValue = make([]byte, 4)
		for i := 0; i < 4; i++ {
			hashValue[i] = byte(sum >> (24 - 8*i))
		}
	case `md5`:
		h := md5.New()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `sha-1`:
		h := sha1.New()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `sha-256`:
		h := sha256.New()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `sha-512`:
		h := sha512.New()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `sha-3-512`:
		h := sha3.New512()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		hashValue = h.Sum([]byte{})
	case `shake256-128`:
		h := sha3.NewShake256()
		_, err := h.Write(hashContent)
		if err != nil {
			return []byte{}, err
		}
		var buf [128]byte
		_, err = h.Read(buf[:])
		hashValue = make([]byte, 16)
		copy(hashValue[:], buf[:])
	}
	return []byte{}, fmt.Errorf(`unknown hash algorithm %s`, hashAlgo)
}

func digestOfNonDir(path string, hashAlgo string, threeMode bool) (string, error) {
	var err error

	// determine nodetype and content
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	var nodetype string
	var content string
	switch mode := fi.Mode(); {
	case mode&os.ModeDevice != 0:
		content = `device file`
		nodetype = `C`
	case mode&os.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}

		content = `link to ` + target
		nodetype = `L`
	case mode&os.ModeNamedPipe != 0:
		content = `FIFO pipe`
		nodetype = `P`
	case mode&os.ModeSocket != 0:
		content = `UNIX domain socket`
		nodetype = `S`
	case mode.IsDir():
		return "", fmt.Errorf(`you are not supposed to call digestOfNonDir with a directory`)
	default:
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}

		content = string(bytes)
		nodetype = `F`
	}

	// determine basename
	basename := filepath.Base(path)

	// if three-mode
	//   digest(file f) := H(f.nodetype ‖ f.basename ‖ f.content)
	// else
	//   digest(file f) := H(f.content)
	var hashContent []byte
	if threeMode {
		hashContent = make([]byte, len(nodetype)+len(basename)+len(content))
		for _, c := range []byte(nodetype) {
			hashContent = append(hashContent, c)
		}
		for _, c := range []byte(basename) {
			hashContent = append(hashContent, c)
		}
		for _, c := range []byte(content) {
			hashContent = append(hashContent, c)
		}
	} else {
		hashContent = make([]byte, len(content))
		for _, c := range []byte(content) {
			hashContent = append(hashContent, c)
		}
	}

	// get hash instance and hash content
	hashValue, err := hashMe(hashContent, hashAlgo)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hashValue), nil
}

func digestOfDir(path string, hashAlgo string, threeMode bool) (string, error) {
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return "", err
	}

	// collect digests in folder
	digests := make([]string, 0, 16)
	for _, fi := range fis {
		if fi.IsDir() {
			digest, err := digestOfDir(filepath.Join(path, fi.Name()), hashAlgo, threeMode)
			if err != nil {
				return "", err
			}

			digests = append(digests, digest)
		} else {
			digest, err := digestOfNonDir(filepath.Join(path, fi.Name()), hashAlgo, threeMode)
			if err != nil {
				return "", err
			}

			digests = append(digests, digest)
		}
	}

	// compute sum of digests
	digest := make([]byte, 512)
	if len(digests) > 0 {
		bytes := len(digests[0])
		digest = digest[0:bytes]
		for _, d := range digests {
			for i := 0; i < bytes; i++ {
				digest[i] = digest[i] ^ []byte(d)[i]
			}
		}
	}

	// potentially add hash of basename
	if threeMode {
		basename := filepath.Base(path)
		d, err := hashMe([]byte(basename), hashAlgo)
		if err != nil {
			return "", err
		}
		digest = digest[0:len(d)]
		for i := 0; i < len(d); i++ {
			digest[i] = digest[i] ^ []byte(d)[i]
		}
	}

	return hex.EncodeToString(digest), nil
}
