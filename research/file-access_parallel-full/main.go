package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var FILES_TOTAL_SIZE int64

func humanReadableBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	base := float64(size) / float64(div)
	return fmt.Sprintf("%.1f %ciB", base, "KMGTPE"[exp])
}

func readEntireFile(filepath string, wg *sync.WaitGroup) {
	defer wg.Done()
	fd, err := os.Open(filepath)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return
		}
		panic(err)
	}
	defer fd.Close()

	var buffer [1024]byte
	for {
		_, err := fd.Read(buffer[:])
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
	}
}

func useOSWalk(dir string, readFiles bool) {
	var wg sync.WaitGroup
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		FILES_TOTAL_SIZE += info.Size()
		if readFiles {
			wg.Add(1)
			go readEntireFile(path, &wg)
		}
		return nil
	})
	if err != nil {
		panic(fmt.Errorf("error walking the path %q: %v\n", dir, err))
	}
	wg.Wait()
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: test-file-access <dir> <1 if read entire file else 0>")
		os.Exit(1)
	}
	start := time.Now()
	useOSWalk(os.Args[1], os.Args[2] == "1")
	diff := time.Now().Sub(start)
	fmt.Printf("time spent on traversal: %s\n", diff)
	fmt.Printf("total size of files: %s\n", humanReadableBytes(FILES_TOTAL_SIZE))
	a := float64(FILES_TOTAL_SIZE)
	b := float64(diff) / float64(time.Second)
	fmt.Printf("⇒ %s/s\n", humanReadableBytes(int64(a/b)))
}
