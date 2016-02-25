package traversal

import (
	"io/ioutil"
	"path"

	"github.com/meisterluk/dupfiles-go/api"
	"github.com/meisterluk/dupfiles-go/utils"
)

// dfsTraversing implements the Traversing interface using a Depth-First strategy
// using ioutil.ReadDir for directory traversal
func dfsTraversing(conf *api.Config, src *api.Source, parent *api.Entry,
	hash api.HashingAlgorithm, out chan *api.Entry, errChan chan error) {
	dir := path.Join(src.Path, parent.Path)

	// retrieve children
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		errChan <- err
		return
	}
	nodes := make([]*api.Entry, 0, len(entries))

	// folders: traverse recursively
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// determine current path
		relPath := path.Join(parent.Path, entry.Name())

		// create a new node
		node := api.Entry{Base: parent.Base, Path: relPath, Parent: parent}
		node.IsDir = true

		nodes = append(nodes, &node)
		dfsTraversing(conf, src, &node, hash, out, errChan)
	}

	// files: compute hashes for files
	for _, entry := range entries {
		if entry.IsDir() || !entry.Mode().IsRegular() {
			continue
		}

		// determine current path
		absPath := path.Join(dir, entry.Name())
		relPath := path.Join(parent.Path, entry.Name())

		// create a new node
		node := api.Entry{Base: parent.Base, Path: relPath, Parent: parent}
		node.IsDir = false

		nodes = append(nodes, &node)

		// compute hash value
		// hash(file) := hash(file content || file basename)
		err := hash.HashFile(conf.HashSpec, absPath, node.Hash[:])
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

	// folders: propagate hashes and finish nodes
	for _, node := range nodes {
		if node.IsDir {
			// if hash is zero, then no files is contained. Just use hash of subfolder

			// hash(folder) = hash(pre-hash(folder) || hash(folder basename))
			if conf.HashSpec.FolderBasename {
				var folderNameHash [api.HASHSIZE]byte
				err = hash.HashString(path.Base(node.Path), folderNameHash[:])
				if err != nil {
					errChan <- err
					return
				}
				err = hash.HashTwoHashes(node.Hash[:], folderNameHash[:], node.Hash[:])
				if err != nil {
					errChan <- err
					return
				}
				hash.HashDirectory(parent.Hash[:], node.Hash[:])
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
	root.Path = ""

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
	state := make(chan error)

	// collect nodes and errors
	go func(state chan error) {
		for {
			select {
			case subnode := <-subnodes:
				tr.Hashes[subnode.Hash] = subnode
				if subnode == root {
					state <- nil
					return
				}
			case e := <-errors:
				state <- e
				return
			}
		}
	}(state)

	// traverse nodes
	go func(state chan error) {
		dfsTraversing(conf, src, root, algo, subnodes, errors)
		subnodes <- root
		state <- nil
	}(state)

	i := 0
	for {
		err := <-state
		if err == nil {
			i++
			if i == 2 {
				return nil
			}
		} else {
			return err
		}
	}
}
