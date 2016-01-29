package fstree

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/meisterluk/dupfiles-go/types"
)

func find(root *types.FSNode, hash [sha256.Size]byte, result *[]*types.Entry) {
	if bytes.Compare(hash[:], root.Node.Hash[:]) == 0 {
		*result = append(*result, root.Node)
		return
	}
	for _, c := range *root.Children {
		find(&c, hash, result)
	}
}

func dfs(roots []types.FSNode, node *types.FSNode, hashes map[string]map[[sha256.Size]byte]*types.Entry, equiv chan []*types.Entry) {
	eqSet := make([]*types.Entry, 0, 3)
	for _, other := range roots {
		if _, ok := hashes[other.Node.Base][node.Node.Hash]; ok {
			otherNodes := make([]*types.Entry, 0, 1)
			find(&other, node.Node.Hash, &otherNodes)
			for _, on := range otherNodes {
				eqSet = append(eqSet, on)
			}
		}
	}
	if len(eqSet) > 0 {
		equiv <- eqSet
	}
}

// SubtreeMatching traverses roots containing file system nodes with hashes
// and reports any equivalent subtrees to the provided channel
func SubtreeMatching(roots []types.FSNode, hashes map[string]map[[sha256.Size]byte]*types.Entry, equiv chan []*types.Entry) error {
	fmt.Printf("Searching in %d root nodes\n", len(roots))
	for _, root := range roots {
		dfs(roots, &root, hashes, equiv)
	}
	close(equiv)
	return nil
}
