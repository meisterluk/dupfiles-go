package internals

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
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

type match struct {
	node       *hierarchyNode
	reportFile string
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
				errChan <- fmt.Errorf(`Inconsistent configuration: %s uses version %d.x, but %s uses version %d.x`, reportFiles[0], refVersion, reportFile, rep.Head.Version[0])
				return
			}
			if refHashAlgorithm != hashAlgorithm {
				errChan <- fmt.Errorf(`Inconsistent configuration: %s uses '%s', but %s uses '%s'`, reportFiles[0], refHashAlgorithm, reportFile, rep.Head.HashAlgorithm)
				return
			}
			if refBasenameMode != baseName {
				errChan <- fmt.Errorf(`Inconsistent configuration: %s uses basename-mode=%t, but %s uses basename-mode=%t`, reportFiles[0], refBasenameMode, reportFile, baseName)
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
	log.Printf("Step 1 of 4 finished: metadata is consistent: version %d, hash algo %s, and %s mode\n", refVersion, refHashAlgorithm, basenameString)

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
	log.Printf("Step 2 of 4 started: reading all digests into memory\n")
	// TODO it is not difficult to get the actual number of entries, right? ⇒ accurate data/estimate
	log.Printf("total file size %d bytes ⇒ estimated %s of main memory required\n", digestSize, humanReadableBytes(memoryRequired))

	data := NewDigestData(digestSizeI, int(estimatedNumEntries>>8))
	entriesFinished := uint64(0)

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

			idx, _ := data.Add(tail.HashValue)
			idx2, _ := data.IndexOf(tail.HashValue)
			if idx2 != idx {
				panic(fmt.Sprintf("new index %d != index %d", idx, idx2))
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
		if len(data.data[i])%(digestSizeI-1+1) != 0 {
			panic("internal error: digest storage broken")
		}
	}
	// at this point, "data" must only be accessed read-only
	log.Printf("Step 2 of 4 finished: reading all digests into memory\n")

	// Step 3: Build a hierarchical [filesystem] tree per reportFile limited to duplicates.
	//         Nodes are references to data.
	log.Printf("Step 3 of 4 started: build filesystem tree of duplicates\n")
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
			index, _ := data.IndexOf(tail.HashValue)

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
			currentNode.digestIndex = uint(index<<1) | 1
		}
		rep.Close()
		trees = append(trees, rootNode)
	}

	// at this point, "trees" must only be accessed read-only

	// verify that all nodes have been initialized (i.e. have digest in file)
	var verifyTree func(*hierarchyNode, *DigestData, string)
	verifyTree = func(node *hierarchyNode, data *DigestData, reportFile string) {
		if node.digestIndex&1 == 0 {
			// determine full path
			fullPath := ""
			currentNode := node
			for {
				fullPath = fullPath + string(filepath.Separator) + node.basename
				if currentNode == currentNode.parent {
					break
				}
				currentNode = currentNode.parent
			}
			if len(fullPath) > 0 {
				fullPath = fullPath[1:]
			}
			panic(fmt.Sprintf("node '%s' is missing in reportFile '%s'", fullPath, reportFile))
		}
		// traverse into children
		for c := 0; c < len(node.children); c++ {
			verifyTree(&node.children[c], data, reportFile)
		}
	}
	for i := range trees {
		verifyTree(trees[i], data, reportFiles[i])
	}

	// TODO just debug information
	data.Dump()
	var dumpTree func(*hierarchyNode, *DigestData, int)
	dumpTree = func(node *hierarchyNode, data *DigestData, indent int) {
		identation := strings.Repeat("  ", indent)
		basename := node.basename
		if node.basename == "" && indent == 0 {
			basename = "."
		}
		log.Printf("%s%s (parent %p) %s and %d child(ren) and %d duplicates\n",
			identation, basename, node.parent,
			hex.EncodeToString(data.Digest(node.digestFirstByte, int(node.digestIndex>>1))),
			len(node.children),
			data.Duplicates(node.digestFirstByte, int(node.digestIndex>>1)),
		)
		for c := 0; c < len(node.children); c++ {
			dumpTree(&node.children[c], data, indent+1)
		}
	}
	for i := range trees {
		log.Println("<tree>")
		dumpTree(trees[i], data, 0)
		log.Println("</tree>")
	}
	log.Printf("Step 3 of 4 finished: filesystem tree of duplicates was built\n")

	// Step 4: traverse tree in DFS and find highest duplicate nodes in tree to publish them
	log.Printf("Step 4 of 4 started: traverse tree in DFS and find highest duplicate nodes\n")

	// we visit *every node* in DFS
	var wg sync.WaitGroup
	for t, tree := range trees {
		for refNode := range traverseTree(tree, data, digestSizeI) {
			// find all nodes with matching digest
			stopSearch := false
			expectedDuplicates := data.Duplicates(refNode.digestFirstByte, int(refNode.digestIndex>>1))

			if expectedDuplicates == 0 {
				continue
			}

			// declare this digest as disabled
			data.Disable(refNode.digestFirstByte, int(refNode.digestIndex>>1))

			// collect matches
			matches := make([]match, 1, ExpectedMatchesPerNode)
			matches[0].node = refNode
			matches[0].reportFile = reportFiles[t]

			for matchData := range matchTree(trees, reportFiles, refNode, data, digestSizeI, &stopSearch) {
				if refNode == matchData.node {
					continue
				}
				matches = append(matches, matchData)

				if len(matches)-1 == expectedDuplicates && expectedDuplicates != MaxCountInDataStructure {
					stopSearch = true
				}
			}

			// TODO remove debug
			//fmt.Printf("found %d matches, expected %d, for %s with %s%s\n", len(matches), expectedDuplicates+1, refNode.basename, hex.EncodeToString([]byte{refNode.digestFirstByte}), hex.EncodeToString(data[refNode.digestFirstByte][int(refNode.digestIndex>>1)*digestSizeI:int(refNode.digestIndex>>1)*digestSizeI+digestSizeI-1]))

			if len(matches) <= 1 {
				times := fmt.Sprintf("%d", expectedDuplicates+1)
				if expectedDuplicates == MaxCountInDataStructure {
					times = "many"
				}
				panic(fmt.Sprintf("internal error: digest %s occurs %s times but search routine found only %d",
					hex.EncodeToString(data.Digest(refNode.digestFirstByte, int(refNode.digestIndex>>1))),
					times,
					len(matches),
				))
			}

			// bubble up matches and publish equivalence sets
			wg.Add(1)
			go func(refNode *hierarchyNode) {
				defer wg.Done()
				bubbleAndPublish(matches, data, outChan, digestSizeI)
				// TODO mark digest of refNode as disabled such that these nodes are not traversed twice
			}(refNode)

			// <profiling>
			/*fd, err := os.Create("mem.prof")
			if err != nil {
				errChan <- err
				return
			}
			pprof.WriteHeapProfile(fd)
			fd.Close()*/
			// </profiling>

			// TODO which one?
			runtime.GC()
			debug.FreeOSMemory() // 690 MB → 600 MB … made some difference
		}
		log.Printf("finished traversal of every node in %s", reportFiles[t])
	}

	log.Printf("finished traversal, but didn't finish matching yet")
	wg.Wait()
	log.Printf("Step 4 of 4 finished: traversed tree in DFS and found highest duplicate nodes\n")

	// TODO print meaningful statistics
}

// traverseTree traverses the given tree defined by the root node
// and emit all nodes starting with the leaves.
func traverseTree(rootNode *hierarchyNode, data *DigestData, digestSizeI int) <-chan *hierarchyNode {
	outChan := make(chan *hierarchyNode)

	var recur func(*hierarchyNode)
	recur = func(node *hierarchyNode) {
		for i := range node.children {
			child := &node.children[i]

			// do not traverse it, if this digest was already analyzed
			disabled := data.Disabled(child.digestFirstByte, int(child.digestIndex>>1))
			if disabled {
				continue
			}

			// traverse into child
			outChan <- child
			recur(child)
		}
	}

	go func() {
		defer close(outChan)
		recur(rootNode)

		disabledRoot := data.Disabled(rootNode.digestFirstByte, int(rootNode.digestIndex>>1))
		if !disabledRoot {
			outChan <- rootNode
		}
	}()
	return outChan
}

// matchTree traverses all trees and emits any equivalent nodes of refNode.
// NOTE this function traverses all trees simultaneously.
// NOTE this function also returns refNode.
func matchTree(trees []*hierarchyNode, reportFiles []string, refNode *hierarchyNode, data *DigestData, digestSize int, stop *bool) <-chan match {
	var wg sync.WaitGroup
	outChan := make(chan match)

	nodesEqual := func(a, b *hierarchyNode) bool {
		if a.digestFirstByte == b.digestFirstByte {
			rDigestSuffix := data.DigestSuffix(b.digestFirstByte, int(b.digestIndex>>1))
			cDigestSuffix := data.DigestSuffix(a.digestFirstByte, int(a.digestIndex>>1))

			if len(rDigestSuffix) != len(cDigestSuffix) {
				panic(fmt.Sprintf("internal error: len(rDigestSuffix) != len(cDigestSuffix); %d != %d", len(rDigestSuffix), len(cDigestSuffix)))
			}
			if eqByteSlices(rDigestSuffix, cDigestSuffix) {
				return true
			}
		}
		return false
	}

	go func(trees []*hierarchyNode, refNode *hierarchyNode, data *DigestData, digestSize int, stop *bool, outChan chan<- match) {
		defer close(outChan)

		var recur func(*hierarchyNode, string)
		recur = func(node *hierarchyNode, reportFile string) {
			for i := range node.children {
				child := &node.children[i]

				// compare digests
				if nodesEqual(child, refNode) {
					outChan <- match{node: child, reportFile: reportFile}
				}

				// stop if maximum number of matches was reached
				if *stop {
					return
				}

				// traverse into child
				runtime.Gosched()
				recur(child, reportFile)
			}
		}

		for i := range trees {
			if nodesEqual(refNode, trees[i]) {
				outChan <- match{node: trees[i], reportFile: reportFiles[i]}
			}

			wg.Add(1)
			go func(tree *hierarchyNode, reportFile string) {
				recur(tree, reportFile)
				wg.Done()
			}(trees[i], reportFiles[i])
		}

		wg.Wait()
	}(trees, refNode, data, digestSize, stop, outChan)

	return outChan
}

// bubbleAndPublish applies the bubbling algorithm and then publishes the equivalent nodes.
// Bubbling is the act of exchanging matches with their parents if at least two matches
// share the same parent. They are collected in clusters and the algorithm is called
// recursively. A cluster is a set of nodes sharing the same digest.
func bubbleAndPublish(matches []match, data *DigestData, outChan chan<- DuplicateSet, digestSize int) {
	if len(matches) < 2 {
		panic("internal error: there must be at least 2 matches")
	}

	parentClusters := make([][]match, 0, 4)
	allAreSingletons := true
	for _, matchData := range matches {
		parent := (*matchData.node).parent

		added := false
		for _, cluster := range parentClusters {
			if cluster[0].node.digestFirstByte == parent.digestFirstByte && cluster[0].node.digestIndex&1 == 1 && cluster[0].node.digestIndex == parent.digestIndex {
				cluster = append(cluster, match{reportFile: matchData.reportFile, node: parent})
				added = true
				allAreSingletons = false
				break
			}
		}

		if !added {
			parentClusters = append(parentClusters, make([]match, 0, len(matches)))
			parentClusters[len(parentClusters)-1] = append(parentClusters[len(parentClusters)-1], match{
				reportFile: matchData.reportFile,
				node:       parent,
			})
		}
	}

	// none of the parent digests match ⇒ emit all && abort bubbling
	if allAreSingletons {
		publishDuplicates(matches, data, outChan, digestSize)
		return
	}

	// some of the parent digests match ⇒ emit all && recurse with 2-or-more clusters
	publishDuplicates(matches, data, outChan, digestSize)
	for i := 0; i < len(parentClusters); i++ {
		if len(parentClusters[i]) >= 2 {
			bubbleAndPublish(parentClusters[i], data, outChan, digestSize)
		}
	}
}

func publishDuplicates(matches []match, data *DigestData, outChan chan<- DuplicateSet, digestSize int) {
	if len(matches) == 0 {
		panic("internal error: matches is empty")
	}

	// collect digest
	digest := data.Digest(matches[1].node.digestFirstByte, int(matches[1].node.digestIndex>>1)) // TODO why “[1]” though?

	outputs := make([]DupOutput, 0, len(matches))
	for _, matchData := range matches {
		approximatePath := (*matchData.node).basename
		node := matchData.node
		for {
			if node.parent == node {
				break
			}
			node = node.parent
			approximatePath = node.basename + string(filepath.Separator) + approximatePath
		}
		if len(approximatePath) > 0 {
			approximatePath = approximatePath[1:]
		}
		outputs = append(outputs, DupOutput{ReportFile: matchData.reportFile, Path: approximatePath})
	}

	outChan <- DuplicateSet{
		Digest: digest,
		Set:    outputs, // TODO: fetch data from file again?
	}
}
