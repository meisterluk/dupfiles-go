package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var COUNT_FILES int64
var FILES_TOTAL_SIZE int64
var FILES_TO_PROCESS int64

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
	return fmt.Sprintf("%.4f %ciB", base, "KMGTPE"[exp])
}

func readEntireFile(filepath string) {
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

func reader(tasks chan string, done chan bool) {
	for {
		select {
		case path := <-tasks:
			readEntireFile(path)
			atomic.AddInt64(&FILES_TO_PROCESS, -1)
		case <-done:
			return
		}
	}
}

func useOSWalk(dir string, readFiles bool, readers int) {
	tasks := make(chan string)
	done := make(chan bool)

	for i := 0; i < readers; i++ {
		go reader(tasks, done)
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if strings.Contains(err.Error(), "permission denied") {
				return nil
			}
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		COUNT_FILES += 1
		FILES_TOTAL_SIZE += info.Size()
		if readFiles {
			atomic.AddInt64(&FILES_TO_PROCESS, 1)
			tasks <- path
		}
		return nil
	})
	if err != nil {
		panic(fmt.Errorf("error walking the path %q: %v\n", dir, err))
	}

	for FILES_TO_PROCESS > 0 {
		time.Sleep(time.Millisecond)
	}
	for i := 0; i < readers; i++ {
		done <- true
	}
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("usage: test-file-access <dir> <1 if read entire file else 0> <# of readers>")
		os.Exit(1)
	}
	start := time.Now()
	no, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic(err)
	}
	useOSWalk(os.Args[1], os.Args[2] == "1", no)
	diff := time.Now().Sub(start)
	fmt.Printf("%d;;%s;%s;%d\n", no, diff, humanReadableBytes(FILES_TOTAL_SIZE), COUNT_FILES)
	//fmt.Printf("time spent on traversal: %s\n", diff)
	//fmt.Printf("total size of files: %s\n", humanReadableBytes(FILES_TOTAL_SIZE))
	//a := float64(FILES_TOTAL_SIZE)
	//b := float64(diff) / float64(time.Second)
	//fmt.Printf("⇒ %s/s\n", humanReadableBytes(int64(a/b)))
}
