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
	fileChan := make(chan string)
	dirChan := make(chan string)
	doneChan1 := make(chan bool)
	doneChan2 := make(chan bool)
	data := make([]string, 0, 10)
	go func() {
		for path := range fileChan {
			data = append(data, filepath.Base(path))
		}
		doneChan1 <- true
	}()
	go func() {
		for path := range dirChan {
			data = append(data, filepath.Base(path))
		}
		doneChan2 <- true
	}()
	err = Walk(filepath.Join(base, "1"), false, false, []string{}, []string{}, []string{}, fileChan, dirChan)
	if err != nil {
		t.Fatal(err)
	}
	<-doneChan1
	<-doneChan2
	expected := "5,6,3,8,7,4,2,1"
	if !compareSlice(data, strings.Split(expected, ",")) {
		t.Fatalf("Expected %s got %s", expected, strings.Join(data, ","))
	}

	// run BFS
	fileChan = make(chan string)
	dirChan = make(chan string)
	data = data[:0]
	go func() {
		for path := range fileChan {
			data = append(data, filepath.Base(path))
		}
		doneChan1 <- true
	}()
	go func() {
		for path := range fileChan {
			data = append(data, filepath.Base(path))
		}
		doneChan2 <- true
	}()
	err = Walk(filepath.Join(base, "1"), true, false, []string{}, []string{}, []string{}, fileChan, dirChan)
	if err != nil {
		t.Fatal(err)
	}
	<-doneChan1
	<-doneChan2
	expected = "2,5,6,3,8,7,4,1"
	if !compareSlice(data, strings.Split(expected, ",")) {
		t.Fatalf("Expected %s got %s", strings.Join(data, ","), expected)
	}

	// teardown
	os.RemoveAll(base)
}
