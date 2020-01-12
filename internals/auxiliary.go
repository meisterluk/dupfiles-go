package internals

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// contains tests whether the given slice contains a particular string item
func contains(set []string, item string) bool {
	for _, element := range set {
		if item == element {
			return true
		}
	}
	return false
}

// compareSlice determines whether string slices as and bs have the same content
func compareSlice(as, bs []string) bool {
	if len(as) != len(bs) {
		return false
	}
	for i, a := range as {
		if a != bs[i] {
			return false
		}
	}
	return true
}

// compareBytes determines whether bytes slices as and bs have the same content
func compareBytes(as, bs []byte) bool {
	if len(as) != len(bs) {
		return false
	}
	for i, a := range as {
		if a != bs[i] {
			return false
		}
	}
	return true
}

// byteEncode implements the byte encoding defined in the design document
func byteEncode(basename string) string {
	if utf8.ValidString(basename) {
		// only individual characters need to be encoded
		re := regexp.MustCompile(`\\{1,}`)
		basename = re.ReplaceAllString(basename, `\$0`)
		basename = strings.Replace(basename, "\x0A", `\x0A`, -1)
		basename = strings.Replace(basename, "\x0B", `\x0B`, -1)
		basename = strings.Replace(basename, "\x0C", `\x0C`, -1)
		basename = strings.Replace(basename, "\x0D", `\x0D`, -1)
		basename = strings.Replace(basename, "\x85", `\x85`, -1)
		basename = strings.Replace(basename, "\xE2\x80\xA8", `\xE2\x80\xA8`, -1) // U+2028
		basename = strings.Replace(basename, "\xE2\x80\xA9", `\xE2\x80\xA9`, -1) // U+2029
		return basename
	}

	// encode the entire string
	s := []byte(basename)
	encoded := make([]byte, 0, 4*len(s))
	for _, b := range s {
		twoChars := strings.ToUpper(hex.EncodeToString([]byte{b}))
		encoded = append(encoded, '\\', 'x', twoChars[0], twoChars[1])
	}
	return string(encoded)
}

// byteDecode implements the inverse operation for "byteEncode(basename string) string".
func byteDecode(basename string) (string, error) {
	if !utf8.ValidString(basename) {
		return "", fmt.Errorf(`byteDecode requires a valid utf-8 string as argument, got '%q'`, basename)
	}
	var err error

	re := regexp.MustCompile(`\\x(0A|0B|0C|0D|85)`)
	basename = re.ReplaceAllStringFunc(basename, func(match string) string {
		s, e := hex.DecodeString(string(match[2:4]))
		err = e
		return string(s)
	})
	if err != nil {
		return "", fmt.Errorf(`byteDecode got an invalid argument: '%q'`, err.Error())
	}

	re2 := regexp.MustCompile(`\\xE2\\x80\\xA(8|9)`)
	basename = re2.ReplaceAllStringFunc(basename, func(match string) string {
		if match == `\\xE2\\x80\\xA8` {
			return "\xE2\x80\xA8"
		}
		return "\xE2\x80\xA9"
	})

	return basename, fmt.Errorf(`byteDecode got an invalid argument: '%q'`, err.Error())
}

func humanReadableBytes(count uint64) string {
	bytes := float64(count)
	units := []string{"bytes", "KiB", "MiB", "GiB", "TiB", "PiB"}
	for _, unit := range units {
		if bytes < 1024 {
			return fmt.Sprintf(`%.02f %s`, bytes, unit)
		}
		bytes /= 1024
	}
	return fmt.Sprintf(`%.02f EiB`, bytes)
}

// isPermissionError determines whether the given error indicates a permission error
func isPermissionError(err error) bool {
	return errors.Is(err, os.ErrPermission)
}

// determineDepth determines the filepath depth of the given filepath.
// For example `a/b` returns 1 and `d/c/b/a` returns 3.
func determineDepth(path string) uint32 {
	// NOTE  This implementation is presumably very inaccurate.
	//       But there is no cross-platform way in golang to do this.
	p := strings.Trim(path, string(filepath.Separator)) // remove leading/trailing separators
	return uint32(strings.Count(p, string(filepath.Separator)))
}

// determineNodeType obviously determines the node type for a give file represented by its os.FileInfo
func determineNodeType(stat os.FileInfo) byte {
	mode := stat.Mode()
	switch {
	case mode&os.ModeDevice != 0:
		return 'C'
	case mode.IsDir():
		return 'D'
	case mode.IsRegular():
		return 'F'
	case mode&os.ModeSymlink != 0:
		return 'L'
	case mode&os.ModeNamedPipe != 0:
		return 'P'
	case mode&os.ModeSocket != 0:
		return 'S'
	}
	return 'X'
}

// xorByteSlices takes byte slices x and y and updates x with x xor y.
// NOTE assumes x and y have same length.
func xorByteSlices(x, y []byte) {
	for i := 0; i < len(x); i++ {
		x[i] = x[i] ^ y[i]
	}
}
