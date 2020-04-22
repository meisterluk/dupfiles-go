package internals

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
)

// ExpectedMatchesPerNode is an internal argument for the
// preallocation size of expected matches for one node
// during the 'find duplicates' process.
const ExpectedMatchesPerNode = 4

// MeanBytesPerLine gives an empirical statistic:
// one tail line in a report file requires 119 bytes in average
const MeanBytesPerLine = 119

// MaxCountInDataStructure gives the highest count of duplicates
// that can be recorded. Internal value. Not adjustable! (e.g. bit repr is used)
// n ⇒ "there are n duplicates" for n in 0..MaxCountInDataStructure-1
// MaxCountInDataStructure ⇒ "there are MaxCountInDataStructure or more matches"
const MaxCountInDataStructure = 127

// HierarchyNode is a data structure to represent a node
// within the filesystem tree represented in a report file
type HierarchyNode struct {
	basename           string
	hashValueFirstByte byte
	hashValueIndex     uint // NOTE first bit is used to store whether "hashValueIndex" and "hashValueFirstByte" have been initialized
	parent             *HierarchyNode
	children           []HierarchyNode
}

// Match represents equivalent nodes found in a report file
type Match struct {
	// TODO public members
	node       *HierarchyNode
	reportFile string
	reportSep  byte
}

// FindDuplicates finds duplicate nodes in report files. The results are sent to outChan.
// Any errors are sent to errChan. At termination outChan and errChan are closed.
func FindDuplicates(reportFiles []string, outChan chan<- DuplicateSet, errChan chan<- error) {
	// TODO add Output as input argument here?

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
	separators := ""
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

		separators = separators + string(rep.Head.Separator)

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

	if len(separators) != len(reportFiles) {
		panic("internal error: stored wrong number of separators")
	}

	algo, err := HashAlgos{}.FromString(refHashAlgorithm)
	if err != nil {
		errChan <- err
		return
	}
	hashValueSizeI := algo.Instance().OutputSize()
	hashValueSize := uint64(hashValueSizeI)
	basenameString := "basename"
	if !refBasenameMode {
		basenameString = "empty"
	}
	log.Printf("Step 1 of 4 finished: metadata is consistent: version %d, hash algo %s, and %s mode\n", refVersion, refHashAlgorithm, basenameString)

	// Step 2: read all hash values into a byte array called data.
	//   The byte array is a sequence of items with values (hash value suffix ‖ disabled bit ‖ dups count).
	//   hash value suffix: hash value byte array without the first byte (we already dispatched with it, right?)
	//   disabled bit: one bit used later to mark nodes as “already processed”
	//   dups count: 7 bits to count the number of duplicates for this hash value (0b111_1111 means “127 or more”)
	// NOTE "count dups" is "hash value occurences - 1"
	estimatedNumEntries := totalFileSize / uint64(MeanBytesPerLine)
	if estimatedNumEntries < 2 {
		estimatedNumEntries = 2
	}
	memoryRequired := estimatedNumEntries * (hashValueSize + 1)
	log.Printf("Step 2 of 4 started: reading all digests into memory\n")
	// TODO it is not difficult to get the actual number of entries, right? ⇒ accurate data/estimate
	log.Printf("total file size %d bytes ⇒ estimated %s of main memory required\n", hashValueSize, HumanReadableBytes(memoryRequired))

	data := NewDigestData(hashValueSizeI, int(estimatedNumEntries>>8))
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
		if len(data.data[i])%(hashValueSizeI-1+1) != 0 {
			panic("internal error: digest storage broken")
		}
	}
	// at this point, "data" must only be accessed read-only
	log.Printf("Step 2 of 4 finished: reading all digests into memory\n")

	// Step 3: Build a hierarchical [filesystem] tree per reportFile limited to duplicates.
	//         Nodes are references to data.
	log.Printf("Step 3 of 4 started: build filesystem tree of duplicates\n")
	trees := make([]*HierarchyNode, 0, len(reportFiles))

	for i, reportFile := range reportFiles {
		log.Printf("reading '%s' …\n", reportFile)
		rootNode := new(HierarchyNode)
		rootNode.parent = rootNode
		rootNode.children = make([]HierarchyNode, 0, 8)

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
			components := PathSplit(tail.Path, separators[i])
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
					currentNode.children = append(currentNode.children, HierarchyNode{
						basename: component,
						parent:   currentNode,
						children: make([]HierarchyNode, 0, 8),
					})
					currentNode = &currentNode.children[len(currentNode.children)-1]
				}
			}
			currentNode.hashValueFirstByte = tail.HashValue[0]
			// we use the first bit to store whether "hashValueIndex" and "hashValueFirstByte" have been properly initialized.
			// this way we detect whether one node is missing in the report file.
			currentNode.hashValueIndex = uint(index<<1) | 1
		}
		rep.Close()
		trees = append(trees, rootNode)
	}

	// at this point, "trees" must only be accessed read-only

	// verify that all nodes have been initialized (i.e. have digest in file)
	var verifyTree func(*HierarchyNode, *DigestData, string, byte)
	verifyTree = func(node *HierarchyNode, data *DigestData, reportFile string, sep byte) {
		if node.hashValueIndex&1 == 0 {
			// determine full path
			fullPath := ""
			components := make([]string, 16)
			currentNode := node
			for {
				components = append(components, node.basename)
				if currentNode == currentNode.parent {
					break
				}
				currentNode = currentNode.parent
			}
			if len(fullPath) > 0 {
				ReverseStringSlice(components)
				fullPath = PathRestore(components, sep)
			}
			panic(fmt.Sprintf("node '%s' is missing in reportFile '%s'", fullPath, reportFile))
		}
		// traverse into children
		for c := 0; c < len(node.children); c++ {
			verifyTree(&node.children[c], data, reportFile, sep)
		}
	}
	for i := range trees {
		verifyTree(trees[i], data, reportFiles[i], separators[i])
	}

	// TODO just debug information
	data.Dump()
	var dumpTree func(*HierarchyNode, *DigestData, int)
	dumpTree = func(node *HierarchyNode, data *DigestData, indent int) {
		identation := strings.Repeat("  ", indent)
		basename := node.basename
		if node.basename == "" && indent == 0 {
			basename = "."
		}
		log.Printf("%s%s (parent %p) %s and %d child(ren) and %d duplicates\n",
			identation, basename, node.parent,
			data.Hash(node.hashValueFirstByte, int(node.hashValueIndex>>1)).Digest(),
			len(node.children),
			data.Duplicates(node.hashValueFirstByte, int(node.hashValueIndex>>1)),
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
		for refNode := range TraverseTree(tree, data, hashValueSizeI) {
			// find all nodes with matching digest
			var stopSearch int32
			expectedDuplicates := data.Duplicates(refNode.hashValueFirstByte, int(refNode.hashValueIndex>>1))

			if expectedDuplicates == 0 {
				continue
			}

			// declare this hash value as disabled
			data.Disable(refNode.hashValueFirstByte, int(refNode.hashValueIndex>>1))

			// collect matches
			matches := make([]Match, 1, ExpectedMatchesPerNode)
			matches[0].node = refNode
			matches[0].reportFile = reportFiles[t]

			for matchData := range MatchTree(trees, reportFiles, refNode, data, hashValueSizeI, &stopSearch) {
				if refNode == matchData.node {
					continue
				}
				matches = append(matches, matchData)

				if len(matches)-1 == expectedDuplicates && expectedDuplicates != MaxCountInDataStructure {
					// NOTE stopSearch used to be a simple boolean.
					// This is not a data race, because there is one writer
					// and an arbitrary number of readers. And it does not matter
					// if readers read value true too late. stopSearch just stops
					// goroutines sooner and thus saves computation time.
					// Golang -race complains it is a data race. So we make it an atomic operation.
					atomic.StoreInt32(&stopSearch, 1)
				}
			}

			// TODO remove debug
			//log.Printf("found %d matches, expected %d, for %s with %s%s\n", len(matches), expectedDuplicates+1, refNode.basename, hex.EncodeToString([]byte{refNode.hashValueFirstByte}), hex.EncodeToString(data[refNode.hashValueFirstByte][int(refNode.hashValueIndex>>1)*hashValueSizeI:int(refNode.hashValueIndex>>1)*hashValueSizeI+hashValueSizeI-1]))

			if len(matches) <= 1 {
				times := fmt.Sprintf("%d", expectedDuplicates+1)
				if expectedDuplicates == MaxCountInDataStructure {
					times = "many"
				}
				panic(fmt.Sprintf("internal error: digest %s occurs %s times but search routine found only %d",
					data.Hash(refNode.hashValueFirstByte, int(refNode.hashValueIndex>>1)).Digest(),
					times,
					len(matches),
				))
			}

			// bubble up matches and publish equivalence sets
			wg.Add(1)
			go func(refNode *HierarchyNode) {
				defer wg.Done()
				BubbleAndPublish(matches, data, outChan, hashValueSizeI)
				// TODO mark hash value of refNode as disabled such that these nodes are not traversed twice
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

// TraverseTree traverses the given tree defined by the root node
// and emit all nodes starting with the leaves.
func TraverseTree(rootNode *HierarchyNode, data *DigestData, hashValueSizeI int) <-chan *HierarchyNode {
	outChan := make(chan *HierarchyNode)

	var recur func(*HierarchyNode)
	recur = func(node *HierarchyNode) {
		for i := range node.children {
			child := &node.children[i]

			// do not traverse it, if this hash value was already analyzed
			disabled := data.Disabled(child.hashValueFirstByte, int(child.hashValueIndex>>1))
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

		disabledRoot := data.Disabled(rootNode.hashValueFirstByte, int(rootNode.hashValueIndex>>1))
		if !disabledRoot {
			outChan <- rootNode
		}
	}()
	return outChan
}

// MatchTree traverses all trees and emits any equivalent nodes of refNode.
// NOTE this function traverses all trees simultaneously.
// NOTE this function also returns refNode.
func MatchTree(trees []*HierarchyNode, reportFiles []string, refNode *HierarchyNode, data *DigestData, hashValueSize int, stop *int32) <-chan Match {
	var wg sync.WaitGroup
	outChan := make(chan Match)

	nodesEqual := func(a, b *HierarchyNode) bool {
		if a.hashValueFirstByte == b.hashValueFirstByte {
			rHashSuffix := data.HashValueSuffix(b.hashValueFirstByte, int(b.hashValueIndex>>1))
			cHashSuffix := data.HashValueSuffix(a.hashValueFirstByte, int(a.hashValueIndex>>1))

			if len(rHashSuffix) != len(cHashSuffix) {
				panic(fmt.Sprintf("internal error: len(rHashSuffix) != len(cHashSuffix); %d != %d", len(rHashSuffix), len(cHashSuffix)))
			}
			if EqByteSlices(rHashSuffix, cHashSuffix) {
				return true
			}
		}
		return false
	}

	go func(trees []*HierarchyNode, refNode *HierarchyNode, data *DigestData, hashValueSize int, stop *int32, outChan chan<- Match) {
		defer close(outChan)

		var recur func(*HierarchyNode, string)
		recur = func(node *HierarchyNode, reportFile string) {
			for i := range node.children {
				child := &node.children[i]

				// compare digests
				if nodesEqual(child, refNode) {
					outChan <- Match{node: child, reportFile: reportFile}
				}

				// stop if maximum number of matches was reached
				if atomic.LoadInt32(stop) == 1 {
					return
				}

				// traverse into child
				runtime.Gosched()
				recur(child, reportFile)
			}
		}

		for i := range trees {
			if nodesEqual(refNode, trees[i]) {
				outChan <- Match{node: trees[i], reportFile: reportFiles[i]}
			}

			wg.Add(1)
			go func(tree *HierarchyNode, reportFile string) {
				recur(tree, reportFile)
				wg.Done()
			}(trees[i], reportFiles[i])
		}

		wg.Wait()
	}(trees, refNode, data, hashValueSize, stop, outChan)

	return outChan
}

// BubbleAndPublish applies the bubbling algorithm and then publishes the equivalent nodes.
// Bubbling is the act of exchanging matches with their parents if at least two matches
// share the same parent. They are collected in clusters and the algorithm is called
// recursively. A cluster is a set of nodes sharing the same digest.
func BubbleAndPublish(matches []Match, data *DigestData, outChan chan<- DuplicateSet, hashValueSize int) {
	if len(matches) < 2 {
		panic("internal error: there must be at least 2 matches")
	}

	parentClusters := make([][]Match, 0, 4)
	allAreSingletons := true
	for _, matchData := range matches {
		parent := (*matchData.node).parent

		added := false
		for _, cluster := range parentClusters {
			if cluster[0].node.hashValueFirstByte == parent.hashValueFirstByte && cluster[0].node.hashValueIndex&1 == 1 && cluster[0].node.hashValueIndex == parent.hashValueIndex {
				cluster = append(cluster, Match{reportFile: matchData.reportFile, node: parent})
				added = true
				allAreSingletons = false
				break
			}
		}

		if !added {
			parentClusters = append(parentClusters, make([]Match, 0, len(matches)))
			parentClusters[len(parentClusters)-1] = append(parentClusters[len(parentClusters)-1], Match{
				reportFile: matchData.reportFile,
				node:       parent,
			})
		}
	}

	// none of the parent digests match ⇒ emit all && abort bubbling
	if allAreSingletons {
		PublishDuplicates(matches, data, outChan, hashValueSize)
		return
	}

	// some of the parent digests match ⇒ emit all
	anySingletons := false
	for i := range parentClusters {
		if len(parentClusters[i]) == 1 {
			anySingletons = true
		}
	}
	if anySingletons {
		PublishDuplicates(matches, data, outChan, hashValueSize)
	}

	// recurse clusters with 2 or more nodes
	for i := 0; i < len(parentClusters); i++ {
		if len(parentClusters[i]) >= 2 {
			BubbleAndPublish(parentClusters[i], data, outChan, hashValueSize)
		}
	}
}

// PublishDuplicates takes matches, evaluates these matches' nodes and sends it to
// outChan where it will be considered as duplicates.
func PublishDuplicates(matches []Match, data *DigestData, outChan chan<- DuplicateSet, hashValueSize int) {
	if len(matches) == 0 {
		panic("internal error: matches is empty")
	}

	// collect digest
	hashValue := data.Hash(matches[1].node.hashValueFirstByte, int(matches[1].node.hashValueIndex>>1)) // TODO why “[1]” though?

	outputs := make([]DupOutput, 0, len(matches))
	for _, matchData := range matches {
		components := make([]string, 0, 16)
		components = append(components, (*matchData.node).basename)
		node := matchData.node
		for {
			if node.parent == node {
				break
			}
			node = node.parent
			components = append(components, node.basename)
		}
		path := PathRestore(components, matchData.reportSep)
		outputs = append(outputs, DupOutput{ReportFile: matchData.reportFile, Path: path})
	}

	outChan <- DuplicateSet{
		HashValue: hashValue,
		Set:       outputs, // TODO: fetch data from file again?
	}
}
