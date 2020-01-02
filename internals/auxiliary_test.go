package internals

import "testing"

func TestByteEncode(t *testing.T) {
	tests := [][2]string{
		[2]string{"\x41\x42\x43\x44\x2E\x74\x78\x74", `ABCD.txt`},
		[2]string{"\x66\x69\x6C\x65\x0A\x6E\x61\x5C\x6D\x65", `file\x0Ana\\me`},
		[2]string{"\x61\xCD\x62", `\x61\xCD\x62`},
		[2]string{"\x6F\x76\x65\x72\x6C\x6F\x6E\x67\xC0\x97", `\x6F\x76\x65\x72\x6C\x6F\x6E\x67\xC0\x97`},
	}
	for _, test := range tests {
		input, expected := test[0], test[1]
		actual := byteEncode(input)
		if actual != expected {
			t.Errorf("Expected byteEncode(%q) = %q; got %q", input, expected, actual)
		}
	}
}
