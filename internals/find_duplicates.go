package internals

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const ExpectedMatchesPerNode = 4

/*
// lineToDigestFound is a simple routine to extract data from a tail line
// without instantiating an entire parser. line must not contain a line feed
// or carriage return.
func lineToDigestFound(line string, lineNo uint64) (digestFound, error) {
	i := 0

	// read hexadecimal digest
	for i < len(line) && line[i] != ' ' {
		i++
	}
	hexDigest := line[0:i]
	for i < len(line) && line[i] == ' ' {
		i++
	}

	// read node type
	if i == len(line) {
		return digestFound{}, fmt.Errorf(`line '%s' does not look like a tail line`, line)
	}
	nodeType := line[i]
	i++
	for i < len(line) && line[i] == ' ' {
		i++
	}

	// read file size
	sizeStart := i
	for i < len(line) && line[i] != ' ' {
		i++
	}
	if sizeStart == i {
		return digestFound{}, fmt.Errorf(`line '%s' does not look like a tail line`, line)
	}
	fileSize, err := strconv.ParseInt(line[sizeStart:i], 10, 64)
	if err != nil {
		return digestFound{}, fmt.Errorf(`file size is not an integer in line '%s'`, line)
	}
	i++

	// return line data
	digest, err := hex.DecodeString(hexDigest)
	if err != nil {
		return digestFound{}, fmt.Errorf(`could not decode hash value of line '%s'`, line)
	}

	return digestFound{
		0,
		lineNo,
		ReportTailLine{
			HashValue: digest,
			NodeType:  nodeType,
			FileSize:  uint64(fileSize),
			Path:      line[i:len(line)],
		},
	}, nil
}

type digestFound struct {
	ReportIndex int
	LineNo      uint64
	ReportTailLine
}

func findDigestMatchesInFile(fd *os.File, hexDigest string) ([]digestFound, error) {
	results := make([]digestFound, 0, 32)
	hexDigestIndex := 0
	lineNumber := uint64(1)

	var buffer [3072]byte

	// read from the beginning of the file
	_, err := fd.Seek(0, 0)
	if err != nil {
		return []digestFound{}, err
	}

	for {
		// reading data into buffer
		n, err := fd.Read(buffer[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return []digestFound{}, err
		}

		// search digest in buffer/view
		view := buffer[0:n]

		for offset, c := range view {
			if c == '\n' {
				hexDigestIndex = 0
				lineNumber++
			}
			if c != hexDigest[hexDigestIndex] {
				hexDigestIndex = 0
				continue
			}

			hexDigestIndex++
			if hexDigestIndex != len(hexDigest) {
				continue
			}

			hexDigestIndex = 0

			// At this point, we found the digest string in the text file.
			// So we seek to the start of the line and read tail line data.

			// seek to beginning of line
			_, err := fd.Seek(-int64(n)+int64(offset)-int64(len(hexDigest))+1, 1)
			if err != nil {
				return []digestFound{}, err
			}

			// read line
			n2, err := fd.Read(buffer[:])
			if err != nil {
				return []digestFound{}, err
			}

			line := buffer[0:n2]
			eolFound := false
			for length := 0; length < len(line); length++ {
				if line[length] == '\n' || line[length] == '\r' {
					line = buffer[0:length]
					eolFound = true
					break
				}
			}
			if !eolFound {
				// TODO not implemented yet: handle separately, do not throw error
				return []digestFound{}, fmt.Errorf(`line uses more than 2048 bytes - internal error`)
			}

			result, err := lineToDigestFound(string(line), lineNumber)
			if err != nil {
				log.Println(err)
			} else {
				results = append(results, result)
			}

			// continue reading file after this line
			_, err = fd.Seek(-int64(n2)+int64(len(line)), 1)
			if err != nil {
				return []digestFound{}, fmt.Errorf(`searching %s in %s: %s`, hexDigest, fd.Name(), err.Error())
			}
			break // must not continue inside modified view/buffer
		}
	}

	return results, nil
}
*/

// MeanBytesPerLine gives an empirical statistic:
// one tail line in a report file requires 119 bytes in average
const MeanBytesPerLine = 119

// MaxCountInDataStructure gives the highest count of duplicates
// that can be recorded. Internal value. Not adjustable! (e.g. bit repr is used)
// n ⇒ "there are n duplicates" for n in 0..MaxCountInDataStructure-1
// MaxCountInDataStructure ⇒ "there are MaxCountInDataStructure or more matches"
const MaxCountInDataStructure = 127

type hierarchyNode struct {
	basename        string
	digestFirstByte byte
	digestIndex     uint // NOTE first bit is used to store whether "digestIndex" and "digestFirstByte" have been initialized
	parent          *hierarchyNode
	children        []hierarchyNode
}

// DigestData essentially represents a list of all digests occuring at least twice
// TODO use DigestData, not data
type DigestData struct {
	digestSize int // equals the size of one entry in slice data[n]
	data       [256][]byte
}

func (d *DigestData) Digest(firstByte byte, idx int) []byte {
	digest := make([]byte, d.digestSize)
	copy(digest[0:1], []byte{firstByte})
	copy(digest[1:], d.data[firstByte][idx*d.digestSize:(idx+1)*d.digestSize])
	return digest
}

func (d *DigestData) Disabled(firstByte byte, idx int) bool {
	return int(d.data[firstByte][(idx+1)*d.digestSize-1])&(MaxCountInDataStructure+1) > 0
}

func (d *DigestData) Duplicates(firstByte byte, idx int) int {
	return int(d.data[firstByte][(idx+1)*d.digestSize-1]) & MaxCountInDataStructure
}

func (d *DigestData) IndexOf(digest []byte) int {
	digestList := d.data[digest[0]]
	for i := 0; i*d.digestSize < len(digestList); i++ {
		if eqByteSlices(digest[1:], digestList[i*d.digestSize:(i+1)*d.digestSize-1]) {
			return i
		}
	}
	return -1
}

// FindDuplicates finds duplicate nodes in report files. The results are sent to outChan.
// Any errors are sent to errChan. At termination outChan and errChan are closed.
func FindDuplicates(reportFiles []string, outChan chan<- DuplicateSet, errChan chan<- error) {
	defer close(outChan)
	defer close(errChan)

	if len(reportFiles) > 16 {
		errChan <- fmt.Errorf(`Sorry, this implementation does not support more than 16 report files in parallel`)
		close(errChan)
		close(outChan)
		return
	}

	// NOTE thought experiment:
	//   2 of 3 files contain the same digest for a directory node N. Can we skip any subnodes of N?
	//   No, the third [actually not only the third!] node might contain a match in some subdirectory of N.
	// NOTE thought experiment:
	//   all files have the same directory node. Can I skip subentries?
	//   No, there might be more matches.

	// NOTE implementation approaches:

	// (1) Ad quick speedups in special cases
	// APPROACH a shortcut that determines whether the roots of all files are the same
	//   thus, start a second goroutine that checks whether all the digests “.” of the files are the same
	// APPROACH a separate goroutine could read the last thirty directory digests (because they are most likely the most top ones)
	//   and tell whether any are the same. If so, they could give a preinformation.

	// (2) Generic approaches requiring tests [i.e. performance tests for bulk data]
	// APPROACH sort digests ⇒ identifies duplicates by locality
	// APPROACH simply use GNU coreutils sort to sort line
	// APPROACH linear data structure to apply insertion sort ⇒ identifies duplicates by locality
	// APPROACH performance improvement by some map data structures ⇒ amortized constant time lookup
	// APPROACH use file size and MeanBytesPerLine to estimate #(entries) ⇒ dispatch above-mentioned approach based on size

	// (3) Specific substrategies
	// APPROACH find "any duplicate of this digest exists?" first ⇒ reduce the total amount of data
	// APPROACH do not store all digests because it is too much data. Instead pick one digest and a
	//   separate goroutine always has an open file descriptor and seeks to find the digest in the file.
	//   Then the goroutine returns the duplicate's line. Requires good timing balance between requester and seeker.
	// APPROACH create a tree of basenames, yield leaves first and if a matching node is found,
	//   bubble it up until the highest matching parent node is found.
	// APPROACH evaluate within-same-node duplicates first and intra-node duplicates later
	//   → might allow some optimizations? → optimizations yet unclear
	// APPROACH size of hash algorithm tells how much memory might be required
	//   → might allow improved strategy → strategy yet unclear

	// NOTE I think the current implementation looses a lot of time to evaluate the highest node
	//      (build tree → matching → bubbling → reporting). Isn't there some easier way to access the parent?

	var totalFileSize uint64

	// Step 1: check that parameterization is consistent
	var refVersion uint16
	var refHashAlgorithm string
	var refBasenameMode bool
	for _, reportFile := range reportFiles {
		stat, err := os.Stat(reportFile)
		if err != nil {
			errChan <- err
			return
		}
		totalFileSize += uint64(stat.Size())

		rep, err := NewReportReader(reportFile)
		if err != nil {
			errChan <- err
			return
		}
		_, err = rep.Iterate()
		if err != nil {
			errChan <- err
			return
		}
		version := rep.Head.Version[0]
		hashAlgorithm := rep.Head.HashAlgorithm
		baseName := rep.Head.BasenameMode
		rep.Close()

		if refHashAlgorithm == "" {
			refVersion = version
			refHashAlgorithm = hashAlgorithm
			refBasenameMode = baseName
		} else {
			if refVersion != version {
				errChan <- fmt.Errorf(`Found inconsistent version among report files: %d.x as well as %d.x`, refVersion, rep.Head.Version[0])
				return
			}
			if refHashAlgorithm != hashAlgorithm {
				errChan <- fmt.Errorf(`Found inconsistent hash algorithm among report files: %s as well as %s`, refHashAlgorithm, rep.Head.HashAlgorithm)
				return
			}
			if refBasenameMode != baseName {
				errChan <- fmt.Errorf(`Found inconsistent mode among report files: basename mode as well as empty mode`)
				return
			}
		}
	}
	algo, err := HashAlgorithmFromString(refHashAlgorithm)
	if err != nil {
		errChan <- err
		return
	}
	digestSizeI := algo.DigestSize()
	digestSize := uint64(digestSizeI)
	basenameString := "basename"
	if !refBasenameMode {
		basenameString = "empty"
	}
	log.Printf("Step 1 of 5 finished: metadata is consistent: version %d, hash algo %s, and %s mode\n", refVersion, refHashAlgorithm, basenameString)

	// Step 2: read all digests into a byte array called data.
	//   The byte array is a sequence of items with values (digest suffix ‖ disabled bit ‖ dups count).
	//   digest suffix: digest byte array without the first byte (we already dispatched with it, right?)
	//   disabled bit: one bit used later to mark nodes as “already processed”
	//   dups count: 7 bits to count the number of duplicates for this digest (0b111_1111 means “127 or more”)
	// NOTE "count dups" is "digest occurences - 1"
	estimatedNumEntries := totalFileSize / uint64(MeanBytesPerLine)
	if estimatedNumEntries < 2 {
		estimatedNumEntries = 2
	}
	memoryRequired := estimatedNumEntries * (digestSize + 1)
	log.Printf("Step 2 of 5 started: reading all digests into memory\n")
	// TODO it is not difficult to get the actual number of entries, right? ⇒ accurate data/estimate
	log.Printf("total file size %d bytes ⇒ estimated %s of main memory required\n", digestSize, humanReadableBytes(memoryRequired))

	var data [256][]byte
	entriesFinished := uint64(0)
	for i := 0; i < 256; i++ {
		data[i] = make([]byte, 0, estimatedNumEntries>>8)
	}

	percentages := [9]uint64{
		estimatedNumEntries / 10,
		(estimatedNumEntries / 10) * 2,
		(estimatedNumEntries / 10) * 3,
		(estimatedNumEntries / 10) * 4,
		(estimatedNumEntries / 10) * 5,
		(estimatedNumEntries / 10) * 6,
		(estimatedNumEntries / 10) * 7,
		(estimatedNumEntries / 10) * 8,
		(estimatedNumEntries / 10) * 9,
	}

	var totalNumEntries uint64
	var totalUniqueDigests uint64
	for _, reportFile := range reportFiles {
		log.Printf("reading '%s' …\n", reportFile)
		rep, err := NewReportReader(reportFile)
		if err != nil {
			errChan <- err
			return
		}
		for {
			tail, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				rep.Close()
				errChan <- err
				return
			}
			totalNumEntries++

			digestSuffix := tail.HashValue[1:]
			digestList := data[int(tail.HashValue[0])]
			found := false
			// -1+1 ⇒ -1 because of first-byte-dispatch and +1 because of dups-byte
			for i := 0; i*(digestSizeI-1+1) < len(digestList); i++ {
				itemDigestSuffix := digestList[i*(digestSizeI-1+1) : (i+1)*(digestSizeI-1+1)-1]
				if eqByteSlices(itemDigestSuffix, digestSuffix) {
					found = true

					// set dups byte to "min(dups + 1, MaxCountInDataStructure)".
					dups := uint(digestList[i*(digestSizeI-1+1)+digestSizeI-1])
					if dups != MaxCountInDataStructure {
						dups++
					}
					digestList[(i+1)*(digestSizeI-1+1)-1] = byte(dups)
					break
				}
			}
			if !found {
				first := int(tail.HashValue[0])
				data[first] = append(data[first], digestSuffix...)
				data[first] = append(data[first], 0)
				totalUniqueDigests++
				// INVARIANT digests are unique within data ⇒ (first-byte, index) uniquely identifies a digest
			}

			entriesFinished++
			for p, threshold := range percentages {
				if entriesFinished == threshold {
					log.Printf("about %d%% of entries read …\n", p*10+10)
				}
			}
		}
		rep.Close()
	}
	for i := 0; i < 256; i++ {
		// test some invariant
		if len(data[i])%(digestSizeI-1+1) != 0 {
			panic("internal error: digest storage broken")
		}
	}
	// at this point, "data" must only be accessed read-only
	log.Printf("Step 2 of 5 finished: reading all digests into memory\n")

	// Step 3: identify digests occuring at least twice
	// NOTE remove all entries with "dups = 0" and shift successive entries to lowest possible index
	log.Printf("Step 3 of 5 started: remove non-duplicate digests from memory\n")
	finishedNumUniqueDigests := uint64(0)
	numDupDigests := uint64(0)
	for i := 0; i < 256; i++ {
		actualEnd := 0
		for j := 0; j*(digestSizeI-1+1) < len(data[i]); j++ {
			dups := data[i][(j+1)*(digestSizeI-1+1)-1]
			if dups != 0 && j == actualEnd {
				// increment actualEnd, but don't copy data [to same location]
				actualEnd++
			} else if dups != 0 {
				copy(
					data[i][(digestSizeI-1+1)*actualEnd:(digestSizeI-1+1)*(actualEnd+1)],
					data[i][(digestSizeI-1+1)*j:(digestSizeI-1+1)*(j+1)],
				)
				actualEnd++
			}
			finishedNumUniqueDigests++
		}
		numDupDigests += uint64(actualEnd)
		data[i] = data[i][0 : (digestSizeI-1+1)*actualEnd]
		if i%8 == 0 {
			ratio := 100.0 * float64(finishedNumUniqueDigests) / float64(totalUniqueDigests)
			log.Printf("finished %.2f%%\n", ratio)
		}
		// TODO: move this algo into the DataDigest data structure
	}
	if finishedNumUniqueDigests != totalUniqueDigests {
		panic(fmt.Sprintf("internal memory: #(finished unique digests) != #(total unique digests); %d != %d", finishedNumUniqueDigests, totalUniqueDigests))
	}
	for i := 0; i < 256; i++ {
		// test some invariants
		if len(data[i])%(digestSizeI-1+1) != 0 {
			panic("internal error: digest storage after reduction is broken")
		}
		for j := 0; digestSizeI*j < len(data[i]); j++ {
			if data[i][j*(digestSizeI-1+1)+digestSizeI-1]&MaxCountInDataStructure == 0 {
				panic(fmt.Sprintf(
					"internal error: after removing non-duplicates, non-duplicates must not exist (digest %s%s)",
					hex.EncodeToString([]byte{data[i][j*digestSizeI]}),
					hex.EncodeToString(data[i][j*digestSizeI:j*digestSizeI+digestSizeI-1]),
				))
			}
			// TODO remove debugging info
			/*log.Printf("%s%s ⇒ %d occurences\n",
				hex.EncodeToString([]byte{data[i][j*digestSizeI]}),
				hex.EncodeToString(data[i][j*digestSizeI:j*digestSizeI+digestSizeI-1]),
				data[i][j*digestSizeI+digestSizeI-1]&MaxCountInDataStructure,
			)*/
		}
	}
	runtime.GC() // hopefully free some memory
	log.Printf("Step 3 of 5 finished: removed %d non-duplicate digests from memory\n", totalNumEntries-numDupDigests)

	// Step 4: Build a hierarchical [filesystem] tree per reportFile limited to duplicates.
	//         Nodes are references to data.
	log.Printf("Step 4 of 5 started: build filesystem tree of duplicates\n")

	trees := make([]*hierarchyNode, 0, len(reportFiles))
	for _, reportFile := range reportFiles {
		log.Printf("reading '%s' …\n", reportFile)
		rootNode := new(hierarchyNode)
		rootNode.parent = rootNode
		rootNode.children = make([]hierarchyNode, 0, 8)

		rep, err := NewReportReader(reportFile)
		if err != nil {
			errChan <- err
			return
		}
		for {
			tail, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				rep.Close()
				errChan <- err
				return
			}

			// ask data: does this node have a duplicate?
			index := uint(0)
			digestList := data[tail.HashValue[0]]
			for i := 0; i*(digestSizeI-1+1) < len(digestList); i++ {
				if eqByteSlices(tail.HashValue[1:], digestList[i*(digestSizeI-1+1):(i+1)*(digestSizeI-1+1)-1]) {
					index = uint(i) + 1
					break
				}
			}
			if index == 0 {
				continue
			} else {
				index--
			}

			// is duplicate ⇒ add to tree
			components := pathSplit(tail.Path)
			currentNode := rootNode
			for _, component := range components {
				// traverse into correct component
				found := false
				for i := range currentNode.children {
					if currentNode.children[i].basename == component {
						found = true
						currentNode = &currentNode.children[i]
						break
					}
				}
				if !found {
					currentNode.children = append(currentNode.children, hierarchyNode{
						basename: component,
						parent:   currentNode,
						children: make([]hierarchyNode, 0, 8),
					})
					currentNode = &currentNode.children[len(currentNode.children)-1]
				}
			}
			currentNode.digestFirstByte = tail.HashValue[0]
			// we use the first bit to store whether "digestIndex" and "digestFirstByte" have been properly initialized.
			// this way we detect whether one node is missing in the report file.
			currentNode.digestIndex = index<<1 | 1
		}
		rep.Close()
		trees = append(trees, rootNode)
	}
	// at this point, "trees" must only be accessed read-only
	// NOTE not all nodes received a proper index and first byte. Why?
	//   the tree is built only with duplicates, so not every node was filled with data.
	// TODO just debug information
	/*var dumpTree func(*hierarchyNode, *[256][]byte, int)
	dumpTree = func(node *hierarchyNode, data *[256][]byte, indent int) {
		for i := 0; i < indent; i++ {
			fmt.Print("  ")
		}
		fmt.Printf("%s (parent %p) %s%s and %d child(ren) and %d duplicates\n",
			node.basename, node.parent, hex.EncodeToString([]byte{node.digestFirstByte}),
			hex.EncodeToString(data[node.digestFirstByte][int(node.digestIndex>>1)*digestSizeI:int(node.digestIndex>>1)*digestSizeI+digestSizeI-1]),
			len(node.children),
			data[node.digestFirstByte][int(node.digestIndex>>1)*digestSizeI+digestSizeI-1]&MaxCountInDataStructure,
		)
		for c := 0; c < len(node.children); c++ {
			dumpTree(&node.children[c], data, indent+1)
		}
	}
	for i := range trees {
		dumpTree(trees[i], &data, 0)
	}*/
	log.Printf("Step 4 of 5 finished: filesystem tree of duplicates was built\n")

	// Step 5: traverse tree in DFS and find highest duplicate nodes in tree to publish them
	log.Printf("Step 5 of 5 started: traverse tree in DFS and find highest duplicate nodes\n")

	// we visit *every node* in DFS
	var wg sync.WaitGroup
	for t, tree := range trees {
		for refNode := range traverseTree(tree, &data, digestSizeI) {
			matches := make([]*hierarchyNode, 1, ExpectedMatchesPerNode)
			matches[0] = refNode

			// find all nodes with matching digest
			stopSearch := false
			expectedDuplicates := int(data[refNode.digestFirstByte][int(refNode.digestIndex>>1)*(digestSizeI-1+1)+digestSizeI-1]) & MaxCountInDataStructure

			for match := range matchTree(trees, refNode, &data, digestSizeI, &stopSearch) {
				if refNode == match {
					continue
				}
				matches = append(matches, match)

				if len(matches)-1 == expectedDuplicates && expectedDuplicates != MaxCountInDataStructure {
					stopSearch = true
				}
			}

			// TODO remove debug
			//fmt.Printf("found %d matches, expected %d, for %s with %s%s\n", len(matches), expectedDuplicates+1, refNode.basename, hex.EncodeToString([]byte{refNode.digestFirstByte}), hex.EncodeToString(data[refNode.digestFirstByte][int(refNode.digestIndex>>1)*digestSizeI:int(refNode.digestIndex>>1)*digestSizeI+digestSizeI-1]))

			if len(matches) == 1 {
				times := fmt.Sprintf("%d", expectedDuplicates+1)
				if expectedDuplicates == MaxCountInDataStructure {
					times = "many"
				}
				start := int(refNode.digestIndex>>1) * (digestSizeI - 1 + 1)
				digestSuffix := data[refNode.digestFirstByte][start : start+digestSizeI-1]
				panic(fmt.Sprintf("internal error: digest %s%s occurs %s times but search routine found only %d",
					hex.EncodeToString([]byte{refNode.digestFirstByte}),
					hex.EncodeToString(digestSuffix),
					times,
					len(matches),
				))
			}

			// bubble up matches and publish equivalence sets
			wg.Add(1)
			go func(refNode *hierarchyNode) {
				defer wg.Done()
				bubbleAndPublish(matches, &data, outChan, digestSizeI, reportFiles[t])
				data[]
			}(refNode)
		}
		log.Printf("finished traversal of every node in %s", reportFiles[t])
	}

	log.Printf("finished traversal, but didn't finish matching yet")
	wg.Wait()
	log.Printf("Step 5 of 5 finished: traversed tree in DFS and found highest duplicate nodes\n")
}

// traverseTree traverses the given tree defined by the root node
// and emit all nodes starting with the leaves.
func traverseTree(rootNode *hierarchyNode, data *[256][]byte, digestSizeI int) <-chan *hierarchyNode {
	outChan := make(chan *hierarchyNode)

	var recur func(*hierarchyNode)
	recur = func(node *hierarchyNode) {
		for _, child := range node.children {
			// do not traverse it, if this digest was already analyzed
			disabled := int(data[child.digestFirstByte][int(child.digestIndex>>1)*(digestSizeI-1+1)+digestSizeI]) & 128
			if disabled > 0 {
				continue
			}

			// traverse into child
			outChan <- &child
			recur(&child)
		}
	}

	go func() {
		defer close(outChan)
		recur(rootNode)
		outChan <- rootNode
	}()
	return outChan
}

// matchTree traverses all trees and emits any equivalent nodes of refNode.
// NOTE this function traverses all trees simultaneously.
// NOTE this function also returns refNode.
func matchTree(trees []*hierarchyNode, refNode *hierarchyNode, data *[256][]byte, digestSize int, stop *bool) <-chan *hierarchyNode {
	var wg sync.WaitGroup
	outChan := make(chan *hierarchyNode)

	go func(trees []*hierarchyNode, refNode *hierarchyNode, data *[256][]byte, digestSize int, stop *bool, outchan chan<- *hierarchyNode) {
		defer close(outChan)

		var recur func(*hierarchyNode)
		recur = func(node *hierarchyNode) {
			for i := range node.children {
				child := &node.children[i]
				/* TODO
				if child.basename == "pencil-and-paper.svg" {
					fmt.Println("child", child.basename)
					fmt.Println("→ child", child)
					fmt.Println("→ refNode", refNode)
					fmt.Println(child.digestFirstByte == refNode.digestFirstByte)
					fmt.Println(child.digestFirstByte, refNode.digestFirstByte)
					//fmt.Println(rDigestSuffix)
				}*/

				// compare digests
				if child.digestFirstByte == refNode.digestFirstByte {
					// TODO why the f*ck can we not move rDigestSuffix after the defer statement? This triggers some data race. Which one?
					rDigestSuffix := data[refNode.digestFirstByte][int(refNode.digestIndex>>1)*(digestSize-1+1) : int(refNode.digestIndex>>1)*(digestSize-1+1)+digestSize-1]
					cDigestSuffix := data[child.digestFirstByte][int(child.digestIndex>>1)*(digestSize-1+1) : int(child.digestIndex>>1)*(digestSize-1+1)+digestSize-1]
					/* TODO
					if child.basename == "pencil-and-paper.svg" {
						fmt.Println("details of", child.basename)
						fmt.Println(child.basename, refNode.basename, child.basename == refNode.basename)
						fmt.Println(child.digestFirstByte, refNode.digestFirstByte, child.digestFirstByte == refNode.digestFirstByte)
						fmt.Println(child.digestIndex, refNode.digestIndex, child.digestIndex == refNode.digestIndex)
						fmt.Println(child.parent, refNode.parent, child.parent == refNode.parent)
						fmt.Println(child.children, refNode.children)
						fmt.Println(cDigestSuffix)
						fmt.Println(rDigestSuffix)
					}*/

					if len(rDigestSuffix) != len(cDigestSuffix) {
						panic(fmt.Sprintf("internal error: len(rDigestSuffix) != len(cDigestSuffix); %d != %d", len(rDigestSuffix), len(cDigestSuffix)))
					}
					if eqByteSlices(rDigestSuffix, cDigestSuffix) {
						outChan <- child
					}
				}

				// stop if maximum number of matches was reached
				if *stop {
					return
				}

				// traverse into child
				runtime.Gosched()
				recur(child)
			}
		}

		for i := range trees {
			wg.Add(1)
			go func(tree *hierarchyNode) {
				recur(tree)
				wg.Done()
			}(trees[i])
		}

		wg.Wait()
	}(trees, refNode, data, digestSize, stop, outChan)

	return outChan
}

// bubbleAndPublish applies the bubbling algorithm and then publishes the equivalent nodes.
// Bubbling is the act of exchanging matches with their parents if at least two matches
// share the same parent. They are collected in clusters and the algorithm is called
// recursively. A cluster is a set of nodes sharing the same digest.
func bubbleAndPublish(matches []*hierarchyNode, data *[256][]byte, outChan chan<- DuplicateSet, digestSize int, reportFile string) {
	if len(matches) < 2 {
		panic("internal error: there must be at least 2 matches")
	}

	parentClusters := make([][]*hierarchyNode, 0, 4)
	allAreSingletons := true
	for _, match := range matches {
		parent := (*match).parent

		added := false
		for _, cluster := range parentClusters {
			if cluster[0].digestFirstByte == parent.digestFirstByte && cluster[0].digestIndex&1 == 1 && cluster[0].digestIndex == parent.digestIndex {
				cluster = append(cluster, parent)
				added = true
				allAreSingletons = false
				break
			}
		}

		if !added {
			parentClusters = append(parentClusters, make([]*hierarchyNode, 0, len(matches)))
			parentClusters[len(parentClusters)-1] = append(parentClusters[len(parentClusters)-1], parent)
		}
	}

	// none of the parent digests match ⇒ emit all && abort bubbling
	if allAreSingletons {
		publishDuplicates(matches, data, outChan, digestSize, reportFile)
		return
	}

	// some of the parent digests match ⇒ emit all && recurse with 2-or-more clusters
	publishDuplicates(matches, data, outChan, digestSize, reportFile)
	for i := 0; i < len(parentClusters); i++ {
		if len(parentClusters[i]) >= 2 {
			bubbleAndPublish(parentClusters[i], data, outChan, digestSize, reportFile)
		}
	}
}

func publishDuplicates(matches []*hierarchyNode, data *[256][]byte, outChan chan<- DuplicateSet, digestSize int, reportFile string) {
	if len(matches) == 0 {
		panic("internal error: matches is empty")
	}

	// collect digest
	digest := make([]byte, 0, digestSize)
	digest = append(digest, matches[1].digestFirstByte)
	digest = append(digest, data[matches[1].digestFirstByte][int(matches[1].digestIndex>>1)*(digestSize-1+1):int(matches[1].digestIndex>>1)*(digestSize-1+1)+digestSize]...)

	outputs := make([]DupOutput, 0, len(matches))
	for _, match := range matches {
		approximatePath := (*match).basename
		node := match
		for {
			if node.parent == node {
				break
			}
			node = node.parent
			approximatePath = node.basename + string(filepath.Separator) + approximatePath
		}
		outputs = append(outputs, DupOutput{ReportFile: reportFile, Path: approximatePath})
	}

	outChan <- DuplicateSet{
		Digest: digest,
		Set:    outputs, // TODO: fetch data from file again?
	}
}