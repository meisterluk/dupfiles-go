package types

import "crypto/sha256"

// Config stores any configuration necessary to run the application
type Config struct {
	HashAlgorithm string
	Bases         map[string]*Entry
}

// Entry stores data about file system entries
type Entry struct {
	Base          string
	Path          string
	Hash          [sha256.Size]byte
	Parent        *Entry
	ChildrenCount int32
	IsDir         bool
}

// FSNode represents a node in the hierarchical file system structure
type FSNode struct {
	Children *[]FSNode
	Node     *Entry
	Basename string
}
