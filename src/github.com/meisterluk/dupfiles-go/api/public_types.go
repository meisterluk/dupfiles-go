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
	FileContent    bool
	FilePerm       bool
	FileAbsPath    bool
	FileRelPath    bool
	FileBasename   bool
	FileOwner      bool
	FileGroup      bool
	FileExt        bool
	FileSize       bool
	FileMtime      bool
	FileAtime      bool
	FolderBasename bool
}

// Source represents a file system node whose subtree will be retrieved
type Source struct {
	Path string
	Name string
}

// Any checks whether any boolean flag of HashingSpec is set
func (h *HashingSpec) Any() bool {
	return (h.FileAbsPath || h.FileAtime || h.FileBasename || h.FileContent ||
		h.FileExt || h.FileGroup || h.FileMtime || h.FileOwner || h.FilePerm)
}
