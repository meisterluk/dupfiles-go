package api

// FindDuplicates implements the main functionality to detect duplicate
// file system nodes in Sources using the given Config and prints the
// result as array of string to the out channel.
//
// This is the only public API function
type _FindDuplicates func(conf Config, srcs []Source, out chan []string) error
