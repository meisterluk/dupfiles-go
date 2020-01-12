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

func TestXor(t *testing.T) {
	tests := [][3][]byte{
		[3][]byte{[]byte{}, []byte{}, []byte{}},
		[3][]byte{[]byte{1}, []byte{2}, []byte{3}},
		[3][]byte{[]byte{0x42, 0x99}, []byte{0x16, 0x09}, []byte{0x54, 0x90}},
		[3][]byte{[]byte{1, 4, 2}, []byte{4, 1, 16}, []byte{5, 5, 18}},
	}

	for _, test := range tests {
		expected := test[2]
		xorByteSlices(test[0], test[1])
		actual := test[0]
		if len(expected) != len(actual) {
			t.Fatalf(`expected bytes %v, got %v`, expected, actual)
		}
		for i := 0; i < len(expected); i++ {
			if expected[i] != actual[i] {
				t.Fatalf(`expected byte %v at %d, got %v`, expected[i], i, actual[i])
			}
		}
	}
}
