package tests

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const exampleStructure = `
root
  ref
    example-folder
      example.txt C1
      example2.txt C1
  pathological
    content
      A
        a.txt C1
        b.txt C1
      B
        a.txt C1
        b.txt C1
        c.txt C1
        d.txt C1
      C
        folder1
          x.txt C1
          y.txt C1
        folder2
          x.txt C3
          y.txt C3
      D
        parent
          child
            x.txt C1
      E
        parent1
          child
            x.txt C1
            y.txt C3
        parent2
          a.txt C3
          b.txt C1
    three
      A
        example
          example C1
        example2
          example2 C1
      B
        folder
          folder
            a.txt C2
      C
        complete C0
        com C4
`

var exampleContents map[string]string = map[string]string{
	`C0`: ``,
	`C1`: `dupfiles generates rεports
😊
`,
	`C2`: `a.txt`,
	`C3`: `some content`,
	`C4`: `plete`,
}

func createExampleTree(root string) error {
	// specification
	structure := exampleStructure
	contents := exampleContents

	// parse specification
	scanner := bufio.NewScanner(strings.NewReader(structure))
	currentPath := make([]string, 0, 42)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == `` {
			continue
		}

		// determine indentation
		indentation := 0
		i := 0
		for line[i] == ' ' {
			indentation++
			i += 2
		}
		if line[i] == '\t' {
			return fmt.Errorf(`horizontal tab found in line "%s"; tabs are disallowed`, line)
		}

		currentPath = currentPath[0:indentation]
		fmt.Printf("starting with %v at %d\n", currentPath, indentation)

		// file or folder?
		nonEmptyStrings := make([]string, 0, 6)
		for _, s := range strings.Split(line[i:len(line)], ` `) {
			if strings.TrimSpace(s) == "" {
				continue
			}
			nonEmptyStrings = append(nonEmptyStrings, s)
		}

		var isFile bool
		switch len(nonEmptyStrings) {
		case 1:
			isFile = false
		case 2:
			isFile = true
		default:
			return fmt.Errorf(`invalid line "%s" found`, line)
		}

		// if file, write it
		if isFile {
			// parameters
			basename := nonEmptyStrings[0]
			content := contents[nonEmptyStrings[1]]

			// build file path
			args := []string{root}
			for _, component := range currentPath {
				args = append(args, component)
			}
			args = append(args, basename)

			filePath := filepath.Join(args...)
			fd, err := os.Create(filePath)
			if err != nil {
				return err
			}
			_, err = fd.Write([]byte(content))
			if err != nil {
				fd.Close()
				return err
			}
			err = fd.Close()
			if err != nil {
				return err
			}
		}

		// if is folder, create folder structure
		if !isFile {
			// parameters
			basename := nonEmptyStrings[0]

			// build folder path
			args := []string{root}
			for _, component := range currentPath {
				args = append(args, component)
			}
			args = append(args, basename)

			// create folder
			err := os.Mkdir(filepath.Join(args...), 0o755)
			if err != nil {
				return err
			}

			// push folder to stack
			currentPath = append(currentPath, basename)
		}

		fmt.Printf("finished with %v\n", currentPath)
	}

	return nil
}
