package main

import (
	"fmt"
	"log"
	"os"

	"github.com/meisterluk/dupfiles-go/api"
	"github.com/meisterluk/dupfiles-go/match"
	"github.com/meisterluk/dupfiles-go/traversal"
)

func main() {
	var conf api.Config
	conf.HashAlgorithm = "sha256"
	conf.HashSpec.Content = true
	conf.HashSpec.Relpath = true

	bases := make([]api.Source, 0, 5)

	if len(os.Args)%2 == 0 {
		log.Fatal("Error: number of command-line arguments must be even")
	}

	if len(os.Args) == 1 {
		log.Print("usage: dupfiles <name> <path> [<name> <path>]*")
		os.Exit(0)
	}

	// create file system root instances
	for i := 1; i < len(os.Args); i = i + 2 {
		bases = append(bases, api.Source{Path: os.Args[i+1], Name: os.Args[i]})
	}

	// get ready for traversal
	trees := make([]api.Tree, 0, len(bases))
	treePtrs := make([]*api.Tree, 0, 5)
	for _, base := range bases {
		t := api.Tree{}
		trees = append(trees, t)
		treePtrs = append(treePtrs, &t)
		err := traversal.DFSTraverse(&conf, &base, &t)
		if err != nil {
			log.Fatal(err)
		}
	}

	done := make(chan bool)
	eqChan := make(chan []*api.Entry)

	go func() {
		for eq := range eqChan {
			fmt.Printf("{ ")
			for _, e := range eq {
				fmt.Printf(" %s:%s ", e.Base, e.Path)
			}
			fmt.Printf(" }\n")
		}
		done <- true
	}()

	err := match.Match(&conf, treePtrs, eqChan)
	if err != nil {
		log.Fatal(err)
	}
	close(eqChan)
	<-done
}
