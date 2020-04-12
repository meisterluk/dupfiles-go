package internals

import "fmt"

// This module implements a map {digest byte array: uint16} for any output digest size.
// The abstraction is given by a DigestMap interface implemented for all output digest sizes.
// This module exists only because of a lack of generics in Go.

// DigestMap provides an interface to handle a map {digest: uint16}
type DigestMap interface {
	Increment([]byte)
	Count([]byte) uint16
}

type list32 struct{ data map[[32]byte]uint16 }
type list64 struct{ data map[[64]byte]uint16 }
type list128 struct{ data map[[128]byte]uint16 }
type list160 struct{ data map[[160]byte]uint16 }
type list256 struct{ data map[[256]byte]uint16 }
type list512 struct{ data map[[512]byte]uint16 }

func newDigestMap(digestSize int, initAlloc int) (DigestMap, error) {
	switch digestSize {
	case 32:
		l := new(list32)
		l.data = make(map[[32]byte]uint16, initAlloc)
		return l, nil
	case 64:
		l := new(list64)
		l.data = make(map[[64]byte]uint16, initAlloc)
		return l, nil
	case 128:
		l := new(list128)
		l.data = make(map[[128]byte]uint16, initAlloc)
		return l, nil
	case 160:
		l := new(list160)
		l.data = make(map[[160]byte]uint16, initAlloc)
		return l, nil
	case 256:
		l := new(list256)
		l.data = make(map[[256]byte]uint16, initAlloc)
		return l, nil
	case 512:
		l := new(list512)
		l.data = make(map[[512]byte]uint16, initAlloc)
		return l, nil
	default:
		return new(list512), fmt.Errorf(`unknown digest size %d - internal error`, digestSize)
	}
}

func (l *list32) Increment(digest []byte) {
	var key [32]byte
	copy(key[:], digest[0:32])
	l.data[key]++
}

func (l *list32) Count(digest []byte) uint16 {
	var key [32]byte
	copy(key[:], digest[0:32])
	return l.data[key]
}

func (l *list64) Increment(digest []byte) {
	var key [64]byte
	copy(key[:], digest[0:64])
	l.data[key]++
}

func (l *list64) Count(digest []byte) uint16 {
	var key [64]byte
	copy(key[:], digest[0:64])
	return l.data[key]
}

func (l *list128) Increment(digest []byte) {
	var key [128]byte
	copy(key[:], digest[0:128])
	l.data[key]++
}

func (l *list128) Count(digest []byte) uint16 {
	var key [128]byte
	copy(key[:], digest[0:128])
	return l.data[key]
}

func (l *list160) Increment(digest []byte) {
	var key [160]byte
	copy(key[:], digest[0:160])
	l.data[key]++
}

func (l *list160) Count(digest []byte) uint16 {
	var key [160]byte
	copy(key[:], digest[0:160])
	return l.data[key]
}

func (l *list256) Increment(digest []byte) {
	var key [256]byte
	copy(key[:], digest[0:256])
	l.data[key]++
}

func (l *list256) Count(digest []byte) uint16 {
	var key [256]byte
	copy(key[:], digest[0:256])
	return l.data[key]
}

func (l *list512) Increment(digest []byte) {
	var key [512]byte
	copy(key[:], digest[0:512])
	l.data[key]++
}

func (l *list512) Count(digest []byte) uint16 {
	var key [512]byte
	copy(key[:], digest[0:512])
	return l.data[key]
}