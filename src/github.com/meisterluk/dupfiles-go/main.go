package main

import (
	"fmt"
	"log"
	"os"

	"github.com/meisterluk/dupfiles-go/api"
	"github.com/meisterluk/dupfiles-go/run"
)

// Main implements the main routine but is independent of os.Args
func Main(args []string) {
	var conf api.Config
	conf.HashAlgorithm = "sha512"
	conf.HashSpec.Content = true
	conf.HashSpec.Relpath = true

	// create file system root instances
	bases := make([]api.Source, 0, 5)
	for i := 1; i < len(args); i = i + 2 {
		bases = append(bases, api.Source{Path: args[i+1], Name: args[i]})
	}

	done := make(chan bool)
	result := make(chan [][2]string)
	smthg := false

	go func() {
		for eq := range result {
			fmt.Printf("{ ")
			for _, e := range eq {
				fmt.Printf(" %s %s  ", e[0], e[1])
			}
			fmt.Printf("}\n")
			smthg = true
		}
		done <- true
	}()

	err := run.FindDuplicates(conf, bases, result)
	if err != nil {
		log.Fatal(err)
	}

	<-done
	if !smthg {
		fmt.Print("No equivalent nodes found in: ")
		for i := 1; i < len(args); i = i + 2 {
			fmt.Printf("%s ", args[i])
		}
		fmt.Println("")
	}
}

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		log.Println("version 0.1 basic")
		os.Exit(0)
	}

	if len(os.Args)%2 == 0 {
		log.Println("Usage: ./dupfiles <NAME1> <PATH1> [<NAME2> <PATH2>]+")
		log.Fatal("Error: number of command-line arguments must be even")
	}

	if len(os.Args) == 1 {
		log.Print("usage: dupfiles <name> <path> [<name> <path>]*")
		os.Exit(0)
	}

	Main(os.Args)
}
