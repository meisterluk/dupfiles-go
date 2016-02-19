package dupfiles

import (
	"log"
	"sort"
	"strings"
	"testing"

	"github.com/meisterluk/dupfiles-go/api"
	"github.com/meisterluk/dupfiles-go/run"
)

func findDup(t *testing.T, paths map[string]string, expected [][]string) {
	// build sources
	conf := api.Config{}
	srcs := make([]api.Source, 0, 2)
	for k, v := range paths {
		src := api.Source{}
		src.Name = k
		src.Path = v
		srcs = append(srcs, src)
	}

	// matches
	toMatch := make(map[string]bool)
	for _, e := range expected {
		sort.Strings(e)
		sorted := strings.Join(e, ";")
		toMatch[sorted] = false
	}

	// find duplicates
	done := make(chan bool)
	out := make(chan [][2]string)
	go func() {
		for dup := range out {
			elements := make([]string, 0, len(dup))
			for _, e := range dup {
				elements = append(elements, e[0]+":"+e[1])
			}
			sort.Strings(elements)
			sorted := strings.Join(elements, ";")
			if !toMatch[sorted] {
				toMatch[sorted] = true
			} else {
				log.Fatal(sorted + " was already matched - internal error")
			}
		}
		done <- true
	}()
	err := run.FindDuplicates(conf, srcs, out)
	if err != nil {
		t.Error(err)
	}
	<-done

	failed := false
	for id, matched := range toMatch {
		if !matched {
			log.Printf("Did not return the following expected equivalent paths: " + id)
			failed = true
		}
	}

	if failed {
		t.Error("Failed to match all expected equivalent paths")
	}
}

// TestTreeFig1 evaluates testcase fig1
func TestTreeFig1(t *testing.T) {
	paths := map[string]string{"HOME1": "/home/meisterluk", "HOME2": "/home/meisterluk"}
	expected := [][]string{[]string{"HOME1:/home/meisterluk", "HOME2:/home/meisterluk"}}
	findDup(t, paths, expected)
}
