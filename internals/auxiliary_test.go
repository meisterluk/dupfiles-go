package internals

import (
	"fmt"
	"io"
	"testing"
)

func TestContains(t *testing.T) {
	if Contains([]string{"a", "bc"}, "d") {
		t.Errorf("[a, bc] does not contain d")
	}

	if !Contains([]string{"foo", "bar"}, "bar") {
		t.Errorf("[foo, bar] does contain bar")
	}

	if Contains([]string{}, "") {
		t.Errorf("[] does contain an empty string")
	}
}

func TestEqStringSlices(t *testing.T) {
	if eqStringSlices([]string{"foo", "bar"}, []string{"foo"}) {
		t.Errorf("[foo, bar] does equal [foo]")
	}

	if eqStringSlices([]string{"", "abc"}, []string{"abc", ""}) {
		t.Errorf("['', abc] does equal [abc, '']")
	}

	if !eqStringSlices([]string{"foo", "bar"}, []string{"foo", "bar"}) {
		t.Errorf("[foo, bar] does equal [foo, bar]")
	}
}

func TestEqByteSlices(t *testing.T) {
	if !eqByteSlices([]byte("bar"), []byte("bar")) {
		t.Errorf("bar equals bar")
	}

	if eqByteSlices([]byte{'\x00'}, []byte("foo")) {
		t.Errorf(`\x00 does not equal foo`)
	}
}

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

func TestByteDecode(t *testing.T) {
	tests := [][2]string{
		[2]string{"\x41\x42\x43\x44\x2E\x74\x78\x74", `ABCD.txt`},
		[2]string{"\x66\x69\x6C\x65\x0A\x6E\x61\x5C\x6D\x65", `file\x0Ana\\me`},
		[2]string{"\x61\xCD\x62", `\x61\xCD\x62`},
		[2]string{"\x6F\x76\x65\x72\x6C\x6F\x6E\x67\xC0\x97", `\x6F\x76\x65\x72\x6C\x6F\x6E\x67\xC0\x97`},
	}
	for _, test := range tests {
		expected, input := test[0], test[1]
		actual, err := byteDecode(input)
		if err != nil {
			t.Errorf(`byteDecode showed error: %s`, err.Error())
			continue
		}
		if actual != expected {
			t.Errorf("Expected byteDecode(%q) = %q; got %q", input, expected, actual)
		}
	}
}

func TestHumanReadableBytes(t *testing.T) {
	data := map[uint64]string{
		10:      `10.00 bytes`,
		1024:    `1.00 KiB`,
		2097152: `2.00 MiB`,
	}
	for count, repr := range data {
		if humanReadableBytes(count) != repr {
			t.Errorf(`Expected humanReadableBytes(%d) == %q, got %q`, count, repr, humanReadableBytes(count))
		}
	}
}

func TestIsPermissionError(t *testing.T) {
	if isPermissionError(io.EOF) {
		t.Errorf(`io.EOF is not a permission error`)
	}

	if isPermissionError(fmt.Errorf(`hi`)) {
		t.Errorf(`custom error is not a permission error`)
	}
}

func TestDetermineDepth(t *testing.T) {
	data := map[string]uint16{
		`a/b`:     1,
		`d/c/b/a`: 3,
	}
	for path, expected := range data {
		if DetermineDepth(path) != expected {
			t.Errorf(`expected depth of %v is %d, got %d`, path, expected, DetermineDepth(path))
		}
	}
}

func TestXorByteSlices(t *testing.T) {
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
