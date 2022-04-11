package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/meisterluk/dupfiles-go/internals"
	"github.com/spf13/cobra"
)

// DiffCommand defines the CLI command parameters
type DiffCommand struct {
	Nodes        []NodePathPair `json:"nodes"`
	ConfigOutput bool           `json:"config"`
	JSONOutput   bool           `json:"json"`
	Help         bool           `json:"help"`
}

// NodePathPair contain a filesystem node and the report file mentioning it.
// Pairs of these constitute the arguments you need to provide for subcommand diff.
type NodePathPair struct {
	BaseNode string `json:"path"`
	Report   string `json:"report"`
}

// DiffJSONResult is a struct used to serialize JSON output
type DiffJSONResult struct {
	Children []DiffJSONObject `json:"children"`
}

// DiffJSONObject represents one difference match of the diff command
type DiffJSONObject struct {
	Basename string   `json:"basename"`
	Digest   string   `json:"digest"`
	OccursIn []string `json:"occurs-in"`
}

// and is an auxiliary function to help to generate a human-readable list of items
func and(elements []string) string {
	if len(elements) == 0 {
		return ""
	} else if len(elements) == 1 {
		return fmt.Sprintf(`‘%s’`, elements[0])
	}
	elems := make([]string, 0, len(elements))
	for _, e := range elements {
		elems = append(elems, fmt.Sprintf(`‘%s’`, e))
	}
	return fmt.Sprintf(`%s and %s`,
		strings.Join(elems[0:len(elems)-1], ", "),
		elems[len(elems)-1],
	)
}

var diffCommand *DiffCommand

var argPairItems []string

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show difference between filesystem nodes",
	Long: `‘dupfiles diff’ allows the user to compute differences between nodes. A node is specified as a pair of (filepath, report) and given two or more of these pairs, the difference between the given filepaths in the respective report file (based on the metadata such as digest, filesize, basename, and directory structure) is computed. The directory is relative to the root of the report file.
For example:

  dupfiles diff etc/ca-certificates laptop_2020-07-03.fsr
                miscellaneous/etc_backup/ca-certificates nas_xmas_backup.fsr

… returns the child nodes of the two mentioned filepaths and their respective state in comparison to the other filepaths.

‘dupfiles diff’ is currently limited and can only compare filepaths non-recursively. Recursive comparison is planned for future releases, but be aware that visualizations of filesystem differences is difficult. No solutions are known, so no guarantees are provided.
`,
	// Args considers all arguments (in the function arguments and global variables
	// of the command line parser) with the goal to define the global DiffCommand instance
	// called diffCommand and fill it with admissible parameters to run the diff command.
	// It EITHER succeeds, fill diffCommand appropriately and returns nil.
	// OR returns an error instance and diffCommand is incomplete.
	Args: func(cmd *cobra.Command, args []string) error {
		// consider positional arguments as argPairItems
		for _, arg := range args {
			argPairItems = append(argPairItems, arg)
		}

		// create global DiffCommand instance
		diffCommand = new(DiffCommand)
		diffCommand.Nodes = make([]NodePathPair, 0, 8)
		diffCommand.ConfigOutput = argConfigOutput
		diffCommand.JSONOutput = argJSONOutput
		diffCommand.Help = false

		// validate Nodes
		a := argPairItems
		if len(a) == 0 {
			exitCode = 7
			return fmt.Errorf(`At least two [{filepath} {report}] pairs are required for comparison, found %d`, len(a))
		} else if len(a)%2 != 0 {
			exitCode = 7
			return fmt.Errorf(`[{filepath} {report}] pairs required. Thus I expected an even number of arguments, got %d`, len(a))
		}
		for i := 0; i < len(a); i = i + 2 {
			if a[i] == "" {
				a[i] = "."
			} else if a[i] == "" {
				exitCode = 8
				return fmt.Errorf(`empty report filepath for '%s' found; expected a valid filepath`, a[i])
			}
			for len(a[i]) > 0 && a[i][len(a[i])-1] == filepath.Separator {
				a[i] = a[i][:len(a[i])-1]
			}
			diffCommand.Nodes = append(diffCommand.Nodes, NodePathPair{BaseNode: a[i], Report: a[i+1]})
		}

		// handle environment variables
		envJSON, errJSON := EnvToBool("DUPFILES_JSON")
		if errJSON == nil {
			diffCommand.JSONOutput = envJSON
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// NOTE global input variables: {w, log, versionCommand}
		exitCode, cmdError = diffCommand.Run(w, log)
		// NOTE global output variables: {exitCode, cmdError}
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
	argPairItems = make([]string, 0, 8)
	diffCmd.PersistentFlags().StringSliceVar(&argPairItems, `pair-item`, []string{}, `filepath or report item considered for comparison`)
}

func (c *DiffCommand) checkNodePathExistence(w Output, log Output) (int, error) {
	found := make([]bool, len(c.Nodes))

	for t, match := range c.Nodes {
		rep, err := internals.NewReportReader(match.Report)
		if err != nil {
			return 1, err
		}
		for {
			tail, _, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				rep.Close()
				return 9, fmt.Errorf(`failure reading report file '%s' tailline: %s`, match.Report, err)
			}

			// check whether the filepath searched for exists in this FSR file
			if tail.Path == match.BaseNode {
				// TODO this assumes that paths are canonical and do not end with a folder separator
				//   → maybe we can blame this as user's fault?
				//   → since filepath information is now ignored, this should be fine, right?
				found[t] = true
				break
			}
		}
		rep.Close()
	}

	// verify that all requested nodes have been found
	notFound := make([]string, 0, len(c.Nodes))
	any := false
	for i := 0; i < len(c.Nodes); i++ {
		if !found[i] {
			notFound = append(notFound, fmt.Sprintf("'%s' in '%s'", c.Nodes[i].BaseNode, c.Nodes[i].Report))
			any = true
		}
	}
	if any {
		return 8, fmt.Errorf(`failed to find %s`, strings.Join(notFound, " and "))
	}

	return 0, nil
}

func (c *DiffCommand) getUnifiedEntries(w Output, log Output, unifiedEntries *[]string, unifiedSep string) (int, error) {
	// the first set creates the tree
	rep, err := internals.NewReportReader(c.Nodes[0].Report)
	if err != nil {
		return 1, err
	}
	for {
		tail, _, err := rep.Iterate()
		if err == io.EOF {
			break
		}
		if err != nil {
			rep.Close()
			return 9, fmt.Errorf(`failure reading report file '%s' tailline: %s`, c.Nodes[0].Report, err)
		}

		pathSuffix, prefixExists := internals.RemovePathPrefix(tail.Path, c.Nodes[0].BaseNode, rep.Head.Separator)
		if !prefixExists {
			continue
		}
		*unifiedEntries = append(*unifiedEntries, strings.ReplaceAll(pathSuffix, string(rep.Head.Separator), unifiedSep))
	}
	rep.Close()

	// subsequent NodePathPairs will only make the tree smaller
	for _, match := range c.Nodes {
		found := make([]bool, len(*unifiedEntries))

		rep, err := internals.NewReportReader(match.Report)
		if err != nil {
			return 1, err
		}
		for {
			tail, _, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				rep.Close()
				return 9, fmt.Errorf(`failure reading report file '%s' tailline: %s`, match.Report, err)
			}

			normalizedPath := strings.ReplaceAll(tail.Path, string(rep.Head.Separator), unifiedSep)
			pathSuffix, prefixExists := internals.RemovePathPrefix(normalizedPath, match.BaseNode, rep.Head.Separator)
			if !prefixExists {
				continue
			}

			for i, e := range *unifiedEntries {
				if e == pathSuffix {
					found[i] = true
					break
				}
			}
		}
		rep.Close()

		// remove any tree elements which have not been found
		i := 0
		for i < len(found) {
			if !found[i] {
				found[i] = found[len(found)-1]
				// this pattern is more efficient than creating a new list each time
				(*unifiedEntries)[i] = (*unifiedEntries)[len(*unifiedEntries)-1]
				*unifiedEntries = (*unifiedEntries)[:len(*unifiedEntries)-1]
				i -= 1
			}
			i++
		}
	}

	return 0, nil
}

func (c *DiffCommand) unifiedEntriesToTree(w Output, log Output, unifiedEntries *[]string, unifiedSep string, root *node, numOfTrees int) (int, error) {
	sort.Strings(*unifiedEntries)

	for _, e := range *unifiedEntries {
		components := internals.PathSplit(e, unifiedSep[0])

		// go to referred node in tree
		curr := root
		for _, component := range components {
			found := false
			for _, c := range (*curr).Children {
				if c.Basename == component {
					found = true
					curr = c
					break
				}
			}
			if !found {
				newNode := new(node)
				newNode.Basename = component
				newNode.HashValue = make([][]byte, numOfTrees)
				newNode.NodeType = make([]byte, numOfTrees)
				newNode.Size = make([]uint64, numOfTrees)
				newNode.CountChildren = make([]int, numOfTrees)
				newNode.Children = make([]*node, 0, 8)
				curr.Children = append(curr.Children, newNode)
				curr = newNode
			}
		}
	}

	// … then fill in the data.
	for p, pair := range c.Nodes {
		rep, err := internals.NewReportReader(pair.Report)
		if err != nil {
			return 1, err
		}
		for {
			tail, _, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if err != nil {
				rep.Close()
				return 9, fmt.Errorf(`failure reading report file '%s' tailline: %s`, pair.Report, err)
			}

			normalizedPath := strings.ReplaceAll(tail.Path, string(rep.Head.Separator), unifiedSep)
			pathSuffix, prefixExists := internals.RemovePathPrefix(normalizedPath, pair.BaseNode, rep.Head.Separator)
			if !prefixExists {
				continue
			}

			components := internals.PathSplit(pathSuffix, unifiedSep[0])
			var parent *node
			curr := root
			for _, component := range components {
				found := false

				for _, c := range curr.Children {
					if c.Basename == component {
						found = true
						parent = curr
						curr = c
						break
					}
				}

				if !found {
					return 99, fmt.Errorf(`internal error: path '%s' in '%s' found once was not found again`, pair.BaseNode, pair.Report)
				}
			}

			curr.HashValue[p] = tail.HashValue
			curr.NodeType[p] = tail.NodeType
			if parent != nil {
				parent.CountChildren[p] += 1
			}
			curr.Size[p] = tail.Size
		}
		rep.Close()
	}

	return 0, nil
}

// Run executes the CLI command diff on the given parameter set,
// writes the result to Output w and errors/information messages to log.
// It returns a triple (exit code, error)
func (c *DiffCommand) Run(w Output, log Output) (int, error) {
	if c.ConfigOutput {
		// config output is printed in JSON independent of c.JSONOutput
		b, err := json.Marshal(c)
		if err != nil {
			return 6, fmt.Errorf(configJSONErrMsg, err)
		}
		w.Println(string(b))
		return 0, nil
	}

	// (1) we check whether the claimed path actually exists in the associated FSR file
	exitCode, err := c.checkNodePathExistence(w, log)
	if err != nil {
		return exitCode, err
	}

	// (2) we determine the list of unified entries: a file system tree which occurs in all NodePathPairs
	// TODO I would love to create a data structure like “type refNode map[string]([]*refNode, bool)”
	//      (…, …) denotes a tuple/struct and where bool is reused to documents whether this node exists in
	//      consecutive FSR files. However, golang is no good with recursive data structures making this impossible.
	unifiedEntries := make([]string, 0, 128)
	unifiedSep := "/"

	exitCode, err = c.getUnifiedEntries(w, log, &unifiedEntries, "/")
	if err != nil {
		return exitCode, err
	}

	// (3) we build an actual tree from the unified entries
	// first, build the structure …
	numOfTrees := len(c.Nodes)
	root := node{
		Basename:      "",
		HashValue:     make([][]byte, numOfTrees),
		NodeType:      make([]byte, numOfTrees),
		Size:          make([]uint64, numOfTrees),
		CountChildren: make([]int, numOfTrees),
		Children:      make([]*node, 0, 8),
	}

	exitCode, err = c.unifiedEntriesToTree(w, log, &unifiedEntries, unifiedSep, &root, numOfTrees)
	if err != nil {
		return exitCode, err
	}

	//root.Dump(0)

	// (4) iterate tree and determine differences
	changesTree := new(treeOfChanges)
	changesTree.children = make([]*treeOfChanges, 0, len(root.Children))

	exitCode, err = root.toUnifiedTree(changesTree, make([]string, 0, 16), c.Nodes)
	if err != nil {
		return exitCode, err
	}

	/*log.Println("tree to be printed:")
	fmt.Println(changesTree)
	err = changesTree.toString(log, 0)
	if err != nil {
		return 99, err
	}*/

	v := NewTreeVisitor(changesTree)
	v.IterateAndPrint(log)

	/*

		type Identifier struct {
			Digest   string
			BaseName string
		}
		type match []bool
		type matches map[Identifier]match

		diffMatches := make(matches)
		for t, match := range c.Nodes {
			rep, err := internals.NewReportReader(match.Report)
			if err != nil {
				return 1, err
			}
			for {
				tail, _, err := rep.Iterate()
				if err == io.EOF {
					break
				}
				if err != nil {
					rep.Close()
					return 9, fmt.Errorf(`failure reading report file '%s' tailline: %s`, match.Report, err)
				}

				// check whether the filepath searched for exists in this FSR file
				if tail.Path == match.BaseNode && (tail.NodeType == 'D' || tail.NodeType == 'L') {
					// TODO this assumes that paths are canonical and do not end with a folder separator
					//   → maybe we can blame this as user's fault?
					//   → since filepath information is now ignored, this should be fine, right?
					found[t] = true
				}
				if !strings.HasPrefix(tail.Path, match.BaseNode) || internals.DetermineDepth(tail.Path, rep.Head.Separator)-1 != internals.DetermineDepth(match.BaseNode, rep.Head.Separator) {
					continue
				}

				given := Identifier{
					Digest:   internals.Hash(tail.HashValue).Digest(),
					BaseName: internals.Base(tail.Path, rep.Head.Separator),
				}
				value, ok := diffMatches[given]
				if ok {
					value[t] = true
				} else {
					diffMatches[given] = make([]bool, len(c.Nodes))
					diffMatches[given][t] = true
				}
			}
			rep.Close()
		}

		// use the first set to determine the entire set

		fmt.Printf("diffMatches = %v\n", diffMatches)

		if c.JSONOutput {
			data := DiffJSONResult{Children: make([]DiffJSONObject, 0, len(diffMatches))}
			for id, diffMatch := range diffMatches {
				occurences := make([]string, 0, len(c.Nodes))
				for i, matches := range diffMatch {
					if matches {
						occurences = append(occurences, c.Nodes[i].Report)
					}
				}
				data.Children = append(data.Children, DiffJSONObject{
					Basename: id.BaseName,
					Digest:   hex.EncodeToString([]byte(id.Digest)),
					OccursIn: occurences,
				})
			}

			jsonRepr, err := json.Marshal(&data)
			if err != nil {
				return 6, fmt.Errorf(resultJSONErrMsg, err)
			}
			w.Println(string(jsonRepr))

		} else {
			for i, anyMatch := range found {
				if !anyMatch {
					log.Printf("# not found: '%s' in '%s'\n", c.Nodes[i].Report, c.Nodes[i].BaseNode)
				}
			}

			w.Println("")
			w.Println("# '+' means found, '-' means missing")

			for id, diffMatch := range diffMatches {
				for _, matched := range diffMatch {
					if matched {
						w.Printf("+")
					} else {
						w.Printf("-")
					}
				}
				w.Printfln("\t%s\t%s", hex.EncodeToString([]byte(id.Digest)), id.BaseName)
			}
		}*/

	return 0, nil
}

/*
	Visualization ideas:

	┬ root
	├─┬ subroot
	│ └─ [node ‘Linux’ misses file] message.txt
	└─┬ var
		└──┬ www
			├─ [entry not found in nodes ‘FS’ and ‘WIN’]  error_20210628.log
			├─ [files in nodes ‘FS’ and ‘WIN’ are different from node ‘Linux’]  error_latest.log
			└─ [entry not found in node ‘FS’]  error_20210707.log
*/

// Algorithmic idea: do not read entire tree, but only representative nodes of medium size

func showTreeDiff(filepath string, nodes []string, recurseOnEqualMetadata bool) (bool, error) {
	// TODO filepath must have normalized separator
	lookup := func(node string, basepath string, basename string) (internals.ReportTailLine, error) {
		rep, err := internals.NewReportReader(node)
		if err != nil {
			return internals.ReportTailLine{}, err
		}

		path := basepath + string(rep.Head.Separator) + basename
		for {
			tailLine, _, err := rep.Iterate()
			if err == io.EOF {
				break
			}
			if tailLine.Path == path {
				return tailLine, nil
			}
		}

		return internals.ReportTailLine{}, fmt.Errorf(`filepath '%s' not found in basenode '%s'`, filepath, node)
	}

	entries, err := os.ReadDir(filepath)
	if err != nil {
		return false, err
	}
	written := false
	for _, entry := range entries {
		tailLines := make(map[string]internals.ReportTailLine)
		missing := make([]string, 0, 8)

		// collect tail lines
		for _, node := range nodes {
			// find line with `entry` in `node`
			tailLine, err := lookup(node, filepath, entry.Name())
			if strings.Contains(err.Error(), `not found in basenode`) {
				missing = append(missing, node)
			} else if err != nil {
				return false, err
			}

			tailLines[node] = tailLine
		}

		// all metadata uniform?
		uniform := true
	cmp:
		for _, d1 := range tailLines {
			for _, d2 := range tailLines {
				if d1.NodeType != d2.NodeType || d1.Size != d2.Size || bytes.Compare(d1.HashValue, d2.HashValue) != 0 {
					uniform = false
					break cmp
				}
			}
		}

		if uniform && len(missing) == 0 {
			isDirectory := tailLines[nodes[0]].NodeType == 'D'
			if isDirectory && recurseOnEqualMetadata {
				// TODO filepath must have normalized separator
				selectedNodes := make([]string, len(tailLines))
				for node, _ := range tailLines {
					selectedNodes = append(selectedNodes, node)
				}
				w, err := showTreeDiff(filepath+entry.Name(), selectedNodes, recurseOnEqualMetadata)
				if err != nil {
					return false, err
				}
				if w {
					written = true
				}
			}
			continue
		}

		written = true

		if len(missing) != 0 {

		}
	}

	return written, nil
}

// Node represents a node of the unified filesystem tree
// with data from all specified node NodePathPairs
type node struct {
	Basename      string
	HashValue     [][]byte
	NodeType      []byte
	Size          []uint64
	CountChildren []int
	Children      []*node
}

func (n *node) Dump(depth int) {
	fmt.Printf(
		"%s'%s'  %d hashvalues %d nodetypes %d sizes %d children\n",
		strings.Repeat("\t", depth),
		n.Basename,
		len(n.HashValue),
		len(n.NodeType),
		len(n.Size),
		len(n.Children),
	)
	for _, c := range n.Children {
		c.Dump(depth + 1)
	}
}

func (n *node) Eq(first int, second int) bool {
	if bytes.Compare(n.HashValue[first], n.HashValue[second]) != 0 {
		return false
	}
	if n.NodeType[first] != n.NodeType[second] {
		return false
	}
	if n.Size[first] != n.Size[second] {
		return false
	}
	if n.CountChildren[first] != n.CountChildren[second] {
		return false
	}
	return true
}

func (n *node) EqClusters() [][]int {
	clusters := make([][]int, 0, 8)
	k := len(n.NodeType)

	// list of elements not clusterized yet
	remainder := make([]int, k)
	for i := 0; i < k; i++ {
		remainder[i] = i
	}

	// clusterize
rem:
	for r := 0; r < len(remainder); r++ {
		for c := 0; c < len(clusters); c++ {
			if n.Eq(remainder[r], clusters[c][0]) {
				// add to existing cluster
				clusters[c] = append(clusters[c], remainder[r])
				if len(remainder) > 1 {
					remainder[r] = remainder[len(remainder)-1]
				}
				remainder = remainder[:len(remainder)-1]

				continue rem
			}
		}

		// create new cluster entry
		clusters = append(clusters, append(make([]int, 0, 4), remainder[r]))
	}

	return clusters
}

func (n *node) DescribeClusterDifferences(clusters [][]int, reportNames []string) string {
	// ASSUME there is at least one cluster
	// ASSUME each cluster has at least one element

	// auxiliary variable: do all clusters have a length of 1?
	allHaveSizeOne := true
	for _, cluster := range clusters {
		if len(cluster) != 1 {
			allHaveSizeOne = false
			break
		}
	}

	// auxiliary variables: clusterize names
	namesOfCluster := make([][]string, 0, len(clusters))
	for _, cluster := range clusters {
		rs := make([]string, 0, len(cluster))
		for _, j := range cluster {
			rs = append(rs, reportNames[j])
		}
		namesOfCluster = append(namesOfCluster, rs)
	}

	// auxiliary function: give human-readable representation for a node type
	humanReadableNodeType := func(nType byte) string {
		switch nType {
		case 'C':
			return `UNIX device file`
		case 'D':
			return `folder`
		case 'L':
			return `file system link`
		case 'P':
			return `FIFO pipe`
		case 'S':
			return `Unix domain socket`
		default:
			return `regular file`
		}
	}

	switch len(clusters) {
	case 1:
		return "" // empty string indicates that no difference was found

	case 2:
		if n.Size[clusters[0][0]] != n.Size[clusters[1][0]] {
			return fmt.Sprintf(
				`nodes in %s differ from %s in size (%s versus %s)`,
				and(namesOfCluster[0]),
				and(namesOfCluster[1]),
				internals.HumanReadableBytes(n.Size[clusters[0][0]]),
				internals.HumanReadableBytes(n.Size[clusters[1][0]]),
			)
		} else if n.NodeType[clusters[0][0]] != n.NodeType[clusters[1][0]] {
			type0 := humanReadableNodeType(n.NodeType[clusters[0][0]])
			if len(clusters[0]) > 1 {
				type0 = "are " + type0 + "s"
			} else {
				type0 = "is a " + type0
			}

			type1 := humanReadableNodeType(n.NodeType[clusters[1][0]])
			if len(clusters[1]) > 1 {
				type1 = "are " + type1 + "s"
			} else {
				type1 = "is a " + type1
			}

			return fmt.Sprintf(
				`nodes in %s %s whereas %s %s`,
				and(namesOfCluster[0]),
				type0,
				and(namesOfCluster[1]),
				type1,
			)
		}

		return fmt.Sprintf(`nodes in %s differ from %s`, and(namesOfCluster[0]), and(namesOfCluster[1]))

	default:
		diff := "hash value"
		if n.Size[clusters[0][0]] != n.Size[clusters[1][0]] {
			diff = "size"
		} else if n.CountChildren[clusters[0][0]] != n.CountChildren[clusters[1][0]] {
			diff = "number of children"
		} else if n.NodeType[clusters[0][0]] != n.NodeType[clusters[1][0]] {
			diff = "node type"
		}

		if allHaveSizeOne {
			cs := make([]string, 0, len(clusters))
			for i := range clusters {
				cs = append(cs, fmt.Sprintf(`%s`, namesOfCluster[i][0]))
			}

			return fmt.Sprintf(`all nodes from %s differ (e.g. in %s)`, diff, and(cs))

		} else {
			cs := make([]string, 0, len(clusters))
			for i := range clusters {
				cs = append(cs, fmt.Sprintf(`{%s}`, and(namesOfCluster[i])))
			}

			return fmt.Sprintf(`nodes give several clusters (e.g. they differ in %s): %s`, diff, strings.Join(cs, " "))
		}
	}
}

// toUnifiedTree takes a root node for treeOfChanges and generates the
// treeOfChanges based on the unified tree.
// The treeOfChanges is built lazily. So a node is created once a node or its children
// needs to store change data (but not earlier).
func (currentNode *node) toUnifiedTree(baseTreeNode *treeOfChanges, currentPath []string, nodePathPairs []NodePathPair) (int, error) {
	// <describe-differences>
	reports := make([]string, 0, len(nodePathPairs))
	for _, pair := range nodePathPairs {
		reports = append(reports, pair.Report)
	}

	clusters := currentNode.EqClusters()
	changeDesc := currentNode.DescribeClusterDifferences(clusters, reports)
	// </describe-differences>

	if changeDesc != "" {
		// add this description to the responsible node in treeOfChanges.
		// ⇒ traverse from root and create any missing children
		curr := baseTreeNode
		for _, component := range currentPath {
			found := false
			for _, child := range curr.children {
				if child.basename == component {
					found = true
					curr = child
					break
				}
			}
			if !found {
				newNode := new(treeOfChanges)
				newNode.basename = component
				newNode.children = make([]*treeOfChanges, 0, 4)
				newNode.description = ""
				curr.children = append(curr.children, newNode)
				curr = newNode
			}
		}

		curr.description = changeDesc
	}

	for _, child := range currentNode.Children {
		exitCode, err := child.toUnifiedTree(baseTreeNode, append(currentPath, child.Basename), nodePathPairs)
		if err != nil {
			return exitCode, err
		}
	}

	return 0, nil
}

type treeOfChanges struct {
	basename    string
	description string
	children    []*treeOfChanges
}

func NewTreeOfChanges(rootNode string) *treeOfChanges {
	t := new(treeOfChanges)
	t.basename = rootNode
	t.description = ""
	t.children = make([]*treeOfChanges, 0, 4)
	return t
}

func (t *treeOfChanges) Add(path []string, change string) {
	current := t

	for _, item := range path {
		for i := 0; i < len(current.children)-1; i++ {
			if current.children[i].basename == item {
				current = current.children[i]
				break
			} else if current.children[i].basename < item && item < current.children[i+1].basename {
				newChild := new(treeOfChanges)
				newChild.basename = item
				newChild.children = make([]*treeOfChanges, 0, 4)
				tail := current.children[i+1 : len(current.children)]
				current.children = append(current.children[0:i+1], newChild)
				current.children = append(current.children, tail...)
				break
			}
		}

		newChild := new(treeOfChanges)
		newChild.basename = item
		newChild.children = make([]*treeOfChanges, 0, 4)
		current.children = append(current.children, newChild)
		current = newChild
	}

	current.description = change
}

func (t *treeOfChanges) PrintTree() {
	t.PrintNode(t, 0)
	fmt.Println()
}

func (t *treeOfChanges) PrintNode(node *treeOfChanges, depth int) {
	if len(node.children) == 0 {
		fmt.Printf("(%s.%p)", node.basename, node)
	} else {
		fmt.Println("")
		for i := 0; i < depth; i++ {
			fmt.Print(" ")
		}
		fmt.Printf("(%s.%p ", node.basename, t)
		for i, c := range node.children {
			t.PrintNode(c, depth+2)
			if i != len(node.children)-1 {
				fmt.Print(" ")
			}
		}
		fmt.Printf(")")
	}
}

func (t *treeOfChanges) toString(log Output, depth int) error {
	curr := t

	log.Printfln("%s[%s] %s", strings.Repeat("\t", depth), curr.basename, curr.description)

	for _, child := range t.children {
		err := child.toString(log, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

type treeVisitor struct {
	Tree *treeOfChanges
	Path []*treeOfChanges // the nodes in the hierarchy, points to the node which has just been visited
}

func NewTreeVisitor(tree *treeOfChanges) treeVisitor {
	return treeVisitor{
		Tree: tree,
		Path: make([]*treeOfChanges, 0, 12),
	}
}

func nextSibling(child *treeOfChanges, parent *treeOfChanges) *treeOfChanges {
	for i, sibling := range parent.children {
		if sibling == child {
			if 0 <= i+1 && i+1 < len(parent.children) {
				return parent.children[i+1]
			} else {
				return nil
			}
		}
	}
	return nil
}

func (v *treeVisitor) Iterate() (string, bool) {
	// (0) root node? then we don't print any
	if len(v.Path) == 0 {
		if len(v.Tree.children) == 0 {
			return v.Tree.basename, true
		}
		v.Path = append(v.Path, v.Tree.children[0])
		//return v.Tree.children[0].basename, false
	} else

	// (1) advance path
	// (a) has children? pick first child.
	if len(v.Path[len(v.Path)-1].children) > 0 {
		v.Path = append(v.Path, v.Path[len(v.Path)-1].children[0])

	} else {
		for {
			current := v.Path[len(v.Path)-1]
			parent := v.Tree
			if len(v.Path) >= 2 {
				parent = v.Path[len(v.Path)-2]
			}

			// (b) has no children but next sibling? pick next sibling.
			sibling := nextSibling(current, parent)
			if sibling != nil {
				v.Path[len(v.Path)-1] = sibling
				break
			}

			// (c) has no children and no next sibling? move up descendants until next sibling can be found.
			v.Path = v.Path[0 : len(v.Path)-1]
		}
	}

	// (2) determine line prefix
	allLast := true
	isLast := make([]bool, 0, len(v.Path))
	parent := v.Tree
	for _, current := range v.Path {
		if current == parent.children[len(parent.children)-1] {
			isLast = append(isLast, true)
		} else {
			isLast = append(isLast, false)
			allLast = false
		}

		parent = current
	}

	out := ""
	for i := 0; i < len(v.Path); i++ {
		if i == len(v.Path)-1 {
			if isLast[i] {
				out += "└─"
			} else {
				out += "├─"
			}
		} else {
			if isLast[i] {
				out += "  "
			} else {
				out += "│ "
			}
		}
	}
	if len(v.Path[len(v.Path)-1].children) > 0 {
		out += "┬─ "
	} else {
		out += "── "
	}

	// (3) finish line
	current := v.Path[len(v.Path)-1]
	if current.description != "" {
		out += "'" + current.basename + "': " + current.description
	} else {
		out += "'" + current.basename + "'"
	}

	return out, allLast && len(current.children) == 0
}

func (v *treeVisitor) IterateAndPrint(o Output) {
	o.Println(v.Tree.basename)
	if len(v.Tree.children) == 0 {
		return
	}

	for {
		line, wasLastLine := v.Iterate()
		o.Println(line)
		if wasLastLine {
			break
		}
	}
}
