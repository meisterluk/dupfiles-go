package run

import (
	"log"
	"sort"
	"strings"
	"testing"

	"github.com/meisterluk/dupfiles-go/api"
)

func findDup(t *testing.T, paths map[string]string, expected [][]string) {
	// build sources
	conf := api.Config{}
	conf.HashAlgorithm = "sha512"
	conf.HashSpec.FileContent = true
	conf.HashSpec.FileRelPath = true
	conf.HashSpec.FolderBasename = true
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
	err := FindDuplicates(conf, srcs, out)
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
	paths := map[string]string{"Input1": "test_trees/01_fig1/input1", "Input2": "test_trees/01_fig1/input2"}
	expected := [][]string{[]string{"Input1:folder1/folder2/folder3/file_A", "Input2:folder1/folder2/folder3/file_A"}}
	findDup(t, paths, expected)
}

// TestTreeFig2 evaluates testcase fig2
func TestTreeFig2(t *testing.T) {
	paths := map[string]string{"Input1": "test_trees/02_fig2/input1", "Input2": "test_trees/02_fig2/input2"}
	expected := [][]string{[]string{"Input1:", "Input2:"}}
	findDup(t, paths, expected)
}

// TestTreeFig3 evaluates testcase fig3
func TestTreeFig3(t *testing.T) {
	paths := map[string]string{"Input1": "test_trees/03_fig3/input1", "Input2": "test_trees/03_fig3/input2"}
	expected := [][]string{[]string{"Input1:folder1/folder2/folder3/file_B", "Input2:folder1/folder2/folder3/file_B"}}
	findDup(t, paths, expected)
}

// TestTreeFig4 evaluates testcase fig3
func TestTreeFig4(t *testing.T) {
	paths := map[string]string{"Input1": "test_trees/04_fig4/input1", "Input2": "test_trees/04_fig4/input2"}
	expected := [][]string{[]string{"Input1:folder1/folder2/folder3", "Input2:folder1/x/folder3"}}
	findDup(t, paths, expected)
}
