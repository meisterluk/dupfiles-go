package api

// EqChannel defines a channel receiving an array of Entry instance representing
// equivalent file system entries
type EqChannel chan<- []*Entry

// Traversing is a function traversing a hierarchy of file system entries
type Traversing func(conf *Config, src *Source, root *Entry, out chan *Entry) error

// Traverse retrieves traversal results and returns them as a tree
type Traverse func(conf *Config, src *Source, tr *Tree) error

// Match takes Tree instances and determines equivalent file system entries
type _Match func(conf *Config, trees []*Tree, eqChan EqChannel) error
