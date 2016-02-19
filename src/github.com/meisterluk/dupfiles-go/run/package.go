package run

import (
	"log"

	"github.com/meisterluk/dupfiles-go/api"
	"github.com/meisterluk/dupfiles-go/match"
	"github.com/meisterluk/dupfiles-go/traversal"
)

// FindDuplicates implements the major routine to determine file system trees
// and match nodes to find equivalent entries. Search and matching can be
// parameterized using the Config argument. Source defines the set of
// search tree bases and out is the channel the results will be sent to.
// Be aware that the channel will be closed once all results have been found.
func FindDuplicates(conf api.Config, srcs []api.Source, out chan [][2]string) error {
	// get ready for traversal
	trees := make([]api.Tree, 0, len(srcs))
	treePtrs := make([]*api.Tree, 0, 5)
	for _, s := range srcs {
		t := api.Tree{}
		trees = append(trees, t)
		treePtrs = append(treePtrs, &t)
		err := traversal.DFSTraverse(&conf, &s, &t)
		if err != nil {
			log.Fatal(err)
		}
	}

	eqChan := make(chan []*api.Entry)
	go func() {
		for eq := range eqChan {
			data := make([][2]string, 0, 10)
			for _, e := range eq {
				data = append(data, [2]string{e.Base, e.Path})
			}
			out <- data
		}
		close(out)
	}()

	err := match.UnorderedMatch(&conf, treePtrs, eqChan)
	if err != nil {
		log.Fatal(err)
	}
	close(eqChan)

	return nil
}
