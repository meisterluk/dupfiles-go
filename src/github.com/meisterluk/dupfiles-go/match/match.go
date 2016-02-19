package match

import "github.com/meisterluk/dupfiles-go/api"

type set struct {
	data map[[api.HASHSIZE]byte]bool
}

func (s *set) Add(key [api.HASHSIZE]byte) {
	s.data[key] = true
}

func (s *set) Has(key [api.HASHSIZE]byte) bool {
	return s.data[key] == true
}

// UnorderedMatch takes a set of trees, determines duplicate nodes in it.
// Results are unordered, hence {a, b} is returned instead of {a, b} and {b, a}.
func UnorderedMatch(conf *api.Config, trees []*api.Tree, eqChan api.EqChannel) error {
	knownSet := set{data: make(map[[api.HASHSIZE]byte]bool)}
	for _, tree := range trees {
		for hash, entry := range tree.Hashes {
			if knownSet.Has(hash) {
				continue
			}
			knownSet.Add(hash)
			eqSet := make([]*api.Entry, 0, 3)
			for _, other := range trees {
				if tree == other {
					continue
				}

				var hashesEquate, parentHashesEquate bool
				e, hashesEquate := other.Hashes[hash]
				if hashesEquate && e.Parent != nil && entry.Parent != nil {
					parentHashesEquate = e.Parent.Hash == entry.Parent.Hash
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
	}
	return nil
}
