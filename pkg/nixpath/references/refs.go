package references

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nix-community/go-nix/pkg/nixbase32"
	"github.com/nix-community/go-nix/pkg/nixpath"
)

const (
	storePrefixLength = len(nixpath.StoreDir) + 1
	refLength         = len(nixbase32.Alphabet) // Store path hash prefix length
)

// ReferenceScanner scans a stream of data for references to store paths to extract run time dependencies.
type ReferenceScanner struct {
	// Map of store path hashes to full store paths.
	hashes map[string]string

	// Set of hits.
	hits map[string]struct{}

	// Buffer for current partial hit.
	buf [refLength]byte

	// How far into buf is currently written.
	n int
}

func NewReferenceScanner(storePathCandidates []string) (*ReferenceScanner, error) {
	var buf [refLength]byte

	hashes := make(map[string]string)

	for _, storePath := range storePathCandidates {
		if !strings.HasPrefix(storePath, nixpath.StoreDir) {
			return nil, fmt.Errorf("missing store path prefix: %s", storePath)
		}

		// Check length is a valid store path length including dashes
		if len(storePath) < len(nixpath.StoreDir)+refLength+3 {
			return nil, fmt.Errorf("invalid store path length: %d for store path '%s'", len(storePath), storePath)
		}

		hashes[storePath[storePrefixLength:storePrefixLength+refLength]] = storePath
	}

	return &ReferenceScanner{
		hits:   make(map[string]struct{}),
		hashes: hashes,
		buf:    buf,
		n:      0,
	}, nil
}

func (r *ReferenceScanner) References() []string {
	paths := make([]string, len(r.hits))

	i := 0

	for hash := range r.hits {
		paths[i] = r.hashes[hash]
		i++
	}

	sort.Strings(paths)

	return paths
}

func (r *ReferenceScanner) Write(s []byte) (int, error) {
	for _, c := range s {
		if !nixbase32.Is(c) {
			r.n = 0

			continue
		}

		r.buf[r.n] = c
		r.n++

		if r.n == refLength {
			hash := string(r.buf[:])
			if _, ok := r.hashes[hash]; ok {
				r.hits[hash] = struct{}{}
			}

			r.n = 0
		}
	}

	return len(s), nil
}
