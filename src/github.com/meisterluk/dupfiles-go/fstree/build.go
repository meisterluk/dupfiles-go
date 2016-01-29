package fstree

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/meisterluk/dupfiles-go/types"
)

func splitDirs(path string, result *[]string) {
	tmp := strings.Split(path, string(filepath.Separator))
	for _, t := range tmp {
		if t != "" {
			*result = append(*result, t)
		}
	}
}

func getRoot(node *types.Entry, levels int) *types.Entry {
	for n := 0; n < levels; n++ {
		node = node.Parent
		if node == nil {
			log.Fatal("parent node is unexpectedly nil")
		}
	}
	return node
}

// Build takes FSEntry instances from a channel and builds a tree of FSNode
// instances representing the hierarchical path structure with roots
func Build(in chan *types.Entry, roots *[]types.FSNode) error {
	for entry := range in {
		var parts []string
		splitDirs(entry.Path, &parts)

		// find root
		var current *types.FSNode
		for _, root := range *roots {
			if root.Node.Path == parts[0] && root.Node.Base == entry.Base {
				current = &root
			}
		}
		if current == nil {
			rootNode := types.FSNode{Basename: parts[0]}
			rootNode.Node = getRoot(entry.Parent, len(parts)-2)
			rootNode.Children = new([]types.FSNode)
			current = &rootNode
			*roots = append(*roots, rootNode)
		}

		// traverse
		for i := 1; i < len(parts); i++ {
			var found bool
			for c := 0; c < len(*current.Children); c++ {
				if (*current.Children)[c].Basename == parts[i] {
					current = &(*current.Children)[c]
					found = true
					break
				}
			}
			if !found {
				newNode := types.FSNode{Node: entry, Basename: parts[i], Children: new([]types.FSNode)}
				*current.Children = append(*current.Children, newNode)
			}
		}
	}
	return nil
}
