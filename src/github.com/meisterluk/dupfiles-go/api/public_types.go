package api

// HASHSIZE defines the maximum digest size in bytes some implemented hash algorithm uses
const HASHSIZE = 64

// Config stores any configuration necessary to run the application
type Config struct {
	HashSpec          HashingSpec
	HashAlgorithm     string
	TraversalStrategy string
}

// HashingSpec specifies which attributes shall be considered in a hash
type HashingSpec struct {
	Content  bool
	Perm     bool
	Abspath  bool
	Relpath  bool
	Basename bool
	Owner    bool
	Group    bool
	Fileext  bool
	Size     bool
	Mtime    bool
	Atime    bool
}

// Source represents a file system node whose subtree will be retrieved
type Source struct {
	Path string
	Name string
}
