package internals

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compareSlice(as, bs []string) bool {
	if len(as) != len(bs) {
		return false
	}
	for i, a := range as {
		if a != bs[i] {
			return false
		}
	}
	return true
}

func TestDFSBFS(t *testing.T) {
	// setup
	base, err := ioutil.TempDir("", "dupfiles-test")
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(base, `1/3`), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(base, `1/4/7`), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	files := []string{`1/2`, `1/3/5`, `1/3/6`, `1/4/7/8`}
	for _, f := range files {
		fd, err := os.Create(filepath.Join(base, f))
		if err != nil {
			t.Fatal(err)
		}
		fd.Close()
	}

	// run DFS
	pathChan := make(chan string)
	doneChan := make(chan bool)
	data := make([]string, 0, 10)
	go func() {
		for path := range pathChan {
			data = append(data, filepath.Base(path))
		}
		doneChan <- true
	}()
	err = Walk(filepath.Join(base, "1"), false, false, []string{}, []string{}, []string{}, pathChan)
	if err != nil {
		t.Fatal(err)
	}
	<-doneChan
	expected := "5,6,3,8,7,4,2,1"
	if !compareSlice(data, strings.Split(expected, ",")) {
		t.Fatalf("Expected %s got %s", expected, strings.Join(data, ","))
	}

	// run BFS
	pathChan = make(chan string)
	data = data[:0]
	go func() {
		for path := range pathChan {
			data = append(data, filepath.Base(path))
		}
		doneChan <- true
	}()
	err = Walk(filepath.Join(base, "1"), true, false, []string{}, []string{}, []string{}, pathChan)
	if err != nil {
		t.Fatal(err)
	}
	<-doneChan
	expected = "2,5,6,3,8,7,4,1"
	if !compareSlice(data, strings.Split(expected, ",")) {
		t.Fatalf("Expected %s got %s", strings.Join(data, ","), expected)
	}

	// teardown
	os.RemoveAll(base)
}
