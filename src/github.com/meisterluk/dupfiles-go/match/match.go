package match

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/meisterluk/dupfiles-go/api"
)

type set struct {
	data map[[api.HASHSIZE]byte]bool
}

func (s *set) Add(key [api.HASHSIZE]byte) {
	s.data[key] = true
}

func (s *set) Has(key [api.HASHSIZE]byte) bool {
	return s.data[key] == true
}

func dumpTree(tree *api.Tree, node *api.Entry, children map[[api.HASHSIZE]byte][][api.HASHSIZE]byte, level int) {
	for i := 0; i < 2*level; i++ {
		fmt.Printf(" ")
	}
	fmt.Printf(" └ ")
	fmt.Printf("%s %s\n", hex.EncodeToString(node.Hash[:]), node.Path)

	for _, child := range children[node.Hash] {
		c := tree.Hashes[child]
		dumpTree(tree, c, children, level+1)
	}
}

// DumpTrees dumps all hash trees for debugging purposes
func DumpTrees(trees []*api.Tree) {
	for _, tree := range trees {
		children := make(map[[api.HASHSIZE]byte][][api.HASHSIZE]byte)
		for hash, entry := range tree.Hashes {
			if entry.Parent == nil {
				continue
			}
			pHash := entry.Parent.Hash
			if entry == entry.Parent || bytes.Compare(pHash[:], hash[:]) == 0 {
				panic("internal error")
			}
			if children[pHash] == nil {
				children[pHash] = make([][api.HASHSIZE]byte, 0)
			}
			children[pHash] = append(children[pHash], hash)
		}
		dumpTree(tree, tree.Root, children, 0)
	}
}

// UnorderedMatch takes a set of trees, determines duplicate nodes in it.
// Results are unordered, hence {a, b} is returned instead of {a, b} and {b, a}.
func UnorderedMatch(conf *api.Config, trees []*api.Tree, eqChan api.EqChannel) error {
	knownSet := set{data: make(map[[api.HASHSIZE]byte]bool)}
	var knownMutex sync.Mutex

	for _, tree := range trees {
		go func(tree *api.Tree) {
			for hash, entry := range tree.Hashes {
				knownMutex.Lock()
				if knownSet.Has(hash) {
					knownMutex.Unlock()
					continue
				}
				knownSet.Add(hash)
				knownMutex.Unlock()
				eqSet := make([]*api.Entry, 0, 3)
				for _, other := range trees {
					if tree == other {
						continue
					}

					var hashesEquate, parentHashesEquate bool
					e, hashesEquate := other.Hashes[hash]
					if hashesEquate && e.Parent != nil && entry.Parent != nil {
						parentHashesEquate = (e.Parent.Hash == entry.Parent.Hash)
					}
					if hashesEquate && !parentHashesEquate {
						eqSet = append(eqSet, e)
					}
				}
				if len(eqSet) > 0 {
					eqSet = append(eqSet, entry)
					eqChan <- eqSet
				}
			}
		}(tree)
	}
	return nil
}
