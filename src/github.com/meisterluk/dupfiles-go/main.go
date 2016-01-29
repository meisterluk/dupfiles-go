package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/meisterluk/dupfiles-go/fstree"
	"github.com/meisterluk/dupfiles-go/traversal"
	"github.com/meisterluk/dupfiles-go/types"
)

func main() {
	conf := types.Config{}
	conf.Bases = make(map[string]*types.Entry)

	if len(os.Args)%2 == 0 {
		log.Fatal("Error: number of command-line arguments must be even")
	}

	if len(os.Args) == 1 {
		log.Print("usage: dupfiles <name> <path> [<name> <path>]*")
		os.Exit(0)
	}

	// create root FSEntry instances
	for i := 1; i < len(os.Args); i = i + 2 {
		name := os.Args[i]
		path := os.Args[i+1]
		rootnode := types.Entry{Base: name, Path: path, Parent: nil, IsDir: true}
		conf.Bases[name] = &rootnode
	}

	var w sync.WaitGroup
	countRoots := len(conf.Bases)
	listChannel := make(chan *types.Entry)

	// traverse root nodes recursively
	traverse := func(w *sync.WaitGroup, listChannel *chan *types.Entry) {
		for name, entry := range conf.Bases {
			w.Add(1)
			go func(name string, entry *types.Entry) {
				defer w.Done()
				err := traversal.DFS(name, entry, *listChannel)
				if err != nil {
					panic(err)
				}
				*listChannel <- entry
			}(name, entry)
		}
	}
	// list files and match subtrees
	container := make(map[string]map[[sha256.Size]byte]*types.Entry)
	builder := make(chan *types.Entry)
	for base := range conf.Bases {
		container[base] = make(map[[sha256.Size]byte]*types.Entry)
	}
	list := func(w *sync.WaitGroup) {
		for countRoots > 0 {
			entry := <-listChannel
			// TODO: synchronize
			container[entry.Base][entry.Hash] = entry
			fmt.Printf("%s:%s %v\n", entry.Base, hex.EncodeToString(entry.Hash[:]), entry.Path)
			builder <- entry

			// TODO: prefer to test whether entry's parent is nil?
			for _, e := range conf.Bases {
				if entry == e {
					countRoots--
				}
			}
		}
		fmt.Println()
		close(listChannel)
		close(builder)
		w.Done()
	}
	/*findDups := func() {
		values := make([]string, len(container), len(container))
		i := 0
		for base := range container {
			values[i] = base
			i++
		}

		for hash, entry := range container[values[0]] {
			fmt.Printf("{")
			fmt.Printf("%s", hex.EncodeToString(entry.Hash[:]))
			fmt.Printf(" -- %s:%s", entry.Base, entry.Path)
			for i := 1; i < len(values); i++ {
				if entry2, ok := container[values[i]][hash]; ok {
					fmt.Printf(" -- %s:%s", entry2.Base, entry2.Path)
				}
			}
			fmt.Printf("}\n")
		}
	}*/

	var roots []types.FSNode
	go func() {
		fstree.Build(builder, &roots)
		w.Done()
	}()

	w.Add(1)
	go list(&w)
	traverse(&w, &listChannel)

	w.Add(1)

	w.Wait()
	fmt.Println("Done retrieving file system information")
	//findDups()
	equivChan := make(chan []*types.Entry)
	go func() {
		contains := func(haystack []*types.Entry, needle *types.Entry) bool {
			for _, e := range haystack {
				if e == needle {
					return true
				}
			}
			return false
		}
		var knownNodes []*types.Entry
		for f := range equivChan {
			if len(f) > 0 && contains(knownNodes, f[0]) {
				continue
			}
			out := "{"
			for _, e := range f {
				out += fmt.Sprintf("%s:%s = ", e.Base, e.Path)
				knownNodes = append(knownNodes, e)
			}
			fmt.Println(out[:len(out)-3] + "}\n")
		}
	}()
	err := fstree.SubtreeMatching(roots, container, equivChan)
	if err != nil {
		log.Fatal(err)
	}
}
