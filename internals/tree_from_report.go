package internals

import (
	"encoding/hex"
	"io"
	"strings"
)

// TreeNode represents a node of the tree
type TreeNode struct {
	Digest   string      `json:"digest"`
	Type     byte        `json:"type"`
	Size     uint64      `json:"size"`
	Basename string      `json:"name"`
	Children []*TreeNode `json:"children"`
}

// TreeFromReport takes a report file and generates the tree
// represented in the report file
func TreeFromReport(reportFile string) (*TreeNode, error) {
	// TODO discuss approach:
	//      currently: reading all data into memory
	//      other: read only basename and byte offset to line, read data from mmap'ed content on the fly

	report, err := NewReportReader(reportFile)
	if err != nil {
		return nil, err
	}

	root := new(TreeNode)
	root.Children = make([]*TreeNode, 0, 1)

	add := func(root *TreeNode, node *TreeNode, parts []string) {
		current := root

	PARTS:
		for _, part := range parts {
			for _, child := range current.Children {
				if child.Basename == part {
					current = child
					continue PARTS
				}
			}
			intermNode := new(TreeNode)
			intermNode.Basename = part
			intermNode.Children = make([]*TreeNode, 0)
			current.Children = append(current.Children, intermNode)
			current = intermNode
		}

		current.Basename = node.Basename
		current.Digest = node.Digest
		current.Size = node.Size
		current.Type = node.Type
	}

	for {
		line, _, err := report.Iterate()
		if err == io.EOF {
			break
		}
		if err != nil {
			return root, err
		}

		parts := strings.Split(line.Path, string(report.Head.Separator))
		if line.Path == "" {
			parts = []string{}
		}

		node := new(TreeNode)
		if len(parts) > 0 {
			node.Basename = parts[len(parts)-1]
		}
		node.Children = make([]*TreeNode, 0)
		node.Digest = hex.EncodeToString(line.HashValue)
		node.Size = line.FileSize
		node.Type = line.NodeType

		add(root, node, parts)
	}

	return root, nil
}
