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
	if EqStringSlices([]string{"foo", "bar"}, []string{"foo"}) {
		t.Errorf("[foo, bar] equals [foo]? Expected no, got yes")
	}

	if EqStringSlices([]string{"", "abc"}, []string{"abc", ""}) {
		t.Errorf("['', abc] equals [abc, '']? Expected no, got yes")
	}

	if !EqStringSlices([]string{"foo", "bar"}, []string{"foo", "bar"}) {
		t.Errorf("[foo, bar] equals [foo, bar]? Expected yes, got no")
	}
}

func TestEqByteSlices(t *testing.T) {
	if !EqByteSlices([]byte("bar"), []byte("bar")) {
		t.Errorf(`"bar" equals "bar"? Expected yes, got no`)
	}

	if EqByteSlices([]byte{'\x00'}, []byte("foo")) {
		t.Errorf(`\x00 equals "foo"? Expected no, got yes`)
	}
}

func TestReverseStringSlice(t *testing.T) {
	empty := []string{}
	a := []string{"a"}
	ab := []string{"a", "b"}
	abc := []string{"a", "b", "c"}
	abcd := []string{"a_", "b_", "c_", "d_"}

	ReverseStringSlice(empty)
	ReverseStringSlice(a)
	ReverseStringSlice(ab)
	ReverseStringSlice(abc)
	ReverseStringSlice(abcd)

	if !EqStringSlices(empty, []string{}) {
		t.Errorf(`reversed([]) == []? Expected yes, got no`)
	}
	if !EqStringSlices(a, []string{"a"}) {
		t.Errorf(`reversed(["a"}) == ["a"}? Expected yes, got no`)
	}
	if !EqStringSlices(ab, []string{"b", "a"}) {
		t.Errorf(`reversed(["a", "b"]) == []? Expected yes, got no`)
	}
	if !EqStringSlices(abc, []string{"c", "b", "a"}) {
		t.Errorf(`reversed(["a", "b", "c"]) == ["c", "b", "a"]? Expected yes, got no`)
	}
	if !EqStringSlices(abcd, []string{"d_", "c_", "b_", "a_"}) {
		t.Errorf(`Expected ["d_", "c_", "b_", "a_"], got %v`, abcd)
	}
}

func TestStringsSet(t *testing.T) {
	if !EqStringSlices(StringsSet([]string{"a", "b"}), []string{"a", "b"}) {
		t.Errorf(`StringsSet(["a", "b"]) == ["a", "b"]? Expected yes, got no`)
	}
	if !EqStringSlices(StringsSet([]string{"a", "b", "a"}), []string{"a", "b"}) {
		t.Errorf(`StringsSet(["a", "b", "a"]) == ["a", "b"]? Expected yes, got no`)
	}
	if !EqStringSlices(StringsSet([]string{"a", "b", "a", "b"}), []string{"a", "b"}) {
		t.Errorf(`StringsSet(["a", "b", "a", "b"]) == ["a", "b"]? Expected yes, got no`)
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
		actual := ByteEncode(input)
		if actual != expected {
			t.Errorf("Expected ByteEncode(%q) = %q; got %q", input, expected, actual)
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
		actual, err := ByteDecode(input)
		if err != nil {
			t.Errorf(`ByteDecode showed error: %s`, err.Error())
			continue
		}
		if actual != expected {
			t.Errorf("Expected ByteDecode(%q) = %q; got %q", input, expected, actual)
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
		if HumanReadableBytes(count) != repr {
			t.Errorf(`Expected HumanReadableBytes(%d) == %q, got %q`, count, repr, HumanReadableBytes(count))
		}
	}
}

func TestIsPermissionError(t *testing.T) {
	if IsPermissionError(io.EOF) {
		t.Errorf(`io.EOF is not a permission error`)
	}

	if IsPermissionError(fmt.Errorf(`hi`)) {
		t.Errorf(`custom error is not a permission error`)
	}
}

func TestDetermineDepth(t *testing.T) {
	data := map[string]uint16{
		`a/b`:     1,
		`d/c/b/a`: 3,
	}
	for path, expected := range data {
		if DetermineDepth(path, '/') != expected {
			t.Errorf(`expected depth of %v is %d, got %d`, path, expected, DetermineDepth(path, '/'))
		}
	}
}

func TestDir(t *testing.T) {
	if Dir("hello/world", '/') != "hello" {
		t.Errorf(`Dir("hello/world", '/') equals "hello"? Expected yes, got no`)
	}
	if Dir("a/b/c/d/", '/') != "a/b/c/d" {
		t.Errorf(`Dir("a/b/c/d/", '/') equals "a/b/c/d"? Expected yes, got no`)
	}
}

func TestBase(t *testing.T) {
	if Base("hello/world", '/') != "world" {
		t.Errorf(`Base("hello/world", '/') equals "world"? Expected yes, got no`)
	}
	if Base("a/b/c/d/", '/') != "" {
		t.Errorf(`Base("a/b/c/d/", '/') equals ""? Expected yes, got no`)
	}
}

func TestPathSplit(t *testing.T) {
	if !EqStringSlices(PathSplit("hello/world", '/'), []string{"hello", "world"}) {
		t.Errorf(`PathSplit("hello/world", '/') equals ["hello", "world"]? Expected yes, got no`)
	}
	if !EqStringSlices(PathSplit("a/b", '/'), []string{"a", "b"}) {
		t.Errorf(`PathSplit("a/b", '/') equals ["a", "b"]? Expected yes, got no`)
	}
	if EqStringSlices(PathSplit("/a/b", '/'), []string{"a", "b"}) {
		t.Errorf(`PathSplit("/a/b", '/') equals ["a", "b"]? Expected no, got yes`)
	}
}

func TestPathRestore(t *testing.T) {
	if PathRestore([]string{"a", "b"}, '/') != "a/b" {
		t.Errorf(`PathRestore(["a", "b"], '/') equals "a/b"? Expected yes, got no`)
	}
	if PathRestore([]string{"", "a", "b"}, '/') != "/a/b" {
		t.Errorf(`PathRestore(["", "a", "b"], '/') equals "/a/b"? Expected yes, got no`)
	}
}

func TestXORByteSlices(t *testing.T) {
	tests := [][3]Hash{
		[3]Hash{Hash{}, Hash{}, Hash{}},
		[3]Hash{Hash{1}, Hash{2}, Hash{3}},
		[3]Hash{Hash{0x42, 0x99}, Hash{0x16, 0x09}, Hash{0x54, 0x90}},
		[3]Hash{Hash{1, 4, 2}, Hash{4, 1, 16}, Hash{5, 5, 18}},
	}

	for _, test := range tests {
		expected := test[2]
		XORByteSlices(test[0], test[1])
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
