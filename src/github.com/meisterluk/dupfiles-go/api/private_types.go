package api

// Entry stores data about file system nodes
type Entry struct {
	Base   string
	Path   string
	Hash   [HASHSIZE]byte
	Parent *Entry
	IsDir  bool
}

// Tree represents a parsed file system subtree
type Tree struct {
	Root   *Entry
	Hashes map[[HASHSIZE]byte]*Entry
}
