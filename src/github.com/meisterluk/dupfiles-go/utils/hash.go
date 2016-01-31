package utils

import (
	"fmt"
	"strings"

	"github.com/meisterluk/dupfiles-go/api"
)

// HashAlgorithm provides a generic interface to instantiate
// hash algorithm implementations
type HashAlgorithm struct {
	Name string
	Fn   api.HashingAlgorithm
	Spec api.HashingSpec
}

// NewHashAlgorithm returns a api.HashingAlgorithm instance based on a given
// configuration.
func NewHashAlgorithm(conf *api.Config) (api.HashingAlgorithm, error) {
	repl := strings.NewReplacer("-", "")
	norm := strings.ToLower(repl.Replace(conf.HashAlgorithm))

	if norm == "sha256" {
		return sha256hashing{}, nil
	}

	return nil, fmt.Errorf("Hash algorithm " + conf.HashAlgorithm + " not found")
}

// Given two hashes parent and child, compute the XOR of them.
// Store the result in parent.
func xorTwoHashes(parent []byte, child []byte) error {
	for i := 0; i < len(parent); i++ {
		parent[i] ^= child[i]
	}
	return nil
}
