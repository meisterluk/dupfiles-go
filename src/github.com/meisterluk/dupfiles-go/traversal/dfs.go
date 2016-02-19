package traversal

import (
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/meisterluk/dupfiles-go/api"
	"github.com/meisterluk/dupfiles-go/utils"
)

// dfsTraversing implements the Traversing interface using a Depth-First strategy
// using ioutil.ReadDir for directory traversal
func dfsTraversing(conf *api.Config, src *api.Source, parent *api.Entry,
	hash api.HashingAlgorithm, out chan *api.Entry, errChan chan error) {
	// retrieve children
	entries, err := ioutil.ReadDir(parent.Path)
	if err != nil {
		errChan <- err
		return
	}
	nodes := make([]*api.Entry, 0, len(entries))

	// folders: traverse recursively
	for _, entry := range entries {
		if !entry.IsDir() || entry.Mode()&os.ModeType != 0 {
			continue
		}

		// determine current path
		currentPath := path.Join(parent.Path, entry.Name())

		// create a new node
		node := api.Entry{Base: parent.Base, Path: currentPath, Parent: parent}
		node.IsDir = true

		nodes = append(nodes, &node)
		dfsTraversing(conf, src, &node, hash, out, errChan)
	}

	// files: compute hashes for files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// determine current path
		currentPath := path.Join(parent.Path, entry.Name())

		// create a new node
		node := api.Entry{Base: parent.Base, Path: currentPath, Parent: parent}
		node.IsDir = false

		nodes = append(nodes, &node)

		// compute hash value
		// hash(file) := hash(file content || file basename)
		err := hash.HashFile(conf.HashSpec, currentPath, node.Hash[:])
		if err != nil {
			errChan <- err
			return
		}

		// propagate hash to parent
		if parent != nil {
			// pre-hash(folder) := pre-hash(folder) ^ hash(file)
			hash.HashDirectory(parent.Hash[:], node.Hash[:])
		}
	}

	// propagate folder hashes and finish nodes
	for _, node := range nodes {
		if node.IsDir {
			// hash(folder) = hash(pre-hash(folder) || hash(folder basename))
			var folderNameHash [api.HASHSIZE]byte
			err = hash.HashString(path.Base(node.Path), folderNameHash[:])
			if err != nil {
				errChan <- err
				return
			}
			err = hash.HashTwoHashes(node.Hash[:], node.Hash[:], folderNameHash[:])
			if err != nil {
				errChan <- err
				return
			}
		}

		out <- node
	}
}

// DFSTraverse builds a file system Tree for a Source using depth-first search
func DFSTraverse(conf *api.Config, src *api.Source, tr *api.Tree) error {
	// define root Entry
	root := &api.Entry{}
	root.Base = src.Name
	root.IsDir = true
	root.Parent = nil
	root.Path = src.Path

	// define tree
	tr.Hashes = make(map[[api.HASHSIZE]byte]*api.Entry)
	tr.Root = root

	// get hash algorithm instance
	algo, err := utils.NewHashAlgorithm(conf)
	if err != nil {
		return err
	}

	subnodes := make(chan *api.Entry)
	errors := make(chan error)

	var wait sync.WaitGroup
	wait.Add(2)

	// collect nodes and errors
	go func() {
		defer wait.Done()
		for {
			select {
			case subnode := <-subnodes:
				tr.Hashes[subnode.Hash] = subnode
				if subnode == root {
					return
				}
			case e := <-errors:
				err = e
				return
			}
		}
	}()

	// traverse nodes
	go func() {
		defer wait.Done()
		dfsTraversing(conf, src, root, algo, subnodes, errors)
		subnodes <- root
	}()

	wait.Wait()
	return err
}
