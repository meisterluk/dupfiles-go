package traversal

import (
	"crypto/sha256"
	"io/ioutil"
	"path"

	"github.com/meisterluk/dupfiles-go/hash"
	"github.com/meisterluk/dupfiles-go/types"
)

// DFS implements depth-first search directory traversal
func DFS(name string, parent *types.Entry, out chan *types.Entry) error {
	// I cannot use filepath.Walk here, because the hierarchical structure
	// cannot be easily determined
	entries, err := ioutil.ReadDir(parent.Path)
	if err != nil {
		return err
	}
	nodes := make([]*types.Entry, 0, len(entries))

	// DFS traverse folders
	var childrenCount int32
	for _, entry := range entries {
		if !entry.IsDir() {
			childrenCount++
			continue
		}

		// determine current path
		currentPath := path.Join(parent.Path, entry.Name())

		// create a new node
		node := types.Entry{Base: parent.Base, Path: currentPath, Parent: parent, IsDir: true, ChildrenCount: 0}
		nodes = append(nodes, &node)
		err := DFS(name, &node, out)
		if err != nil {
			return err
		}
	}

	parent.ChildrenCount = childrenCount

	// DFS traverse files: compute hashes for files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// determine current path
		currentPath := path.Join(parent.Path, entry.Name())

		// create a new node
		node := types.Entry{Base: parent.Base, Path: currentPath, Parent: parent, IsDir: false}
		nodes = append(nodes, &node)

		// compute hash value
		// hash(file) := hash(file content || file basename)
		err := hash.SHA256FileHash(node.Hash[:], currentPath)
		if err != nil {
			return err
		}

		// propagate hash to parent
		if parent != nil {
			// pre-hash(folder) := pre-hash(folder) ^ hash(file)
			hash.XORDirHash(parent.Hash[:], node.Hash[:])
		}
	}

	// propagate folder hashes and finish nodes
	for _, node := range nodes {
		if node.IsDir && parent != nil {
			// hash(folder) = hash(pre-hash(folder) || hash(folder basename))
			var folderNameHash [sha256.Size]byte
			err = hash.SHA256String(folderNameHash[:], path.Base(node.Path))
			if err != nil {
				return err
			}
			err = hash.SHA256HashTwoHashes(node.Hash[:], node.Hash[:], folderNameHash[:])
			if err != nil {
				return err
			}
		}

		out <- node
		if parent != nil {
			parent.ChildrenCount--
		}
	}

	return nil
}
