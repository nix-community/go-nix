package binarycache

import (
	"context"
	"strings"

	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/storepath"
)

// ResolveClosure resolves the full dependency closure for the given store
// path hashes. It walks References from each .narinfo, filtering out paths
// that the PathFilter reports as already present. Returns narinfos for all
// missing paths in dependency order (leaves first).
//
// When the filter reports a path as present, its entire sub-closure is
// assumed to be present as well and is not traversed. This matches Nix's
// behaviour: a valid store path implies its closure is complete.
func (c *Client) ResolveClosure(
	ctx context.Context,
	hashes []string,
	filter PathFilter,
) ([]*narinfo.NarInfo, error) {
	fetched := make(map[string]*narinfo.NarInfo)
	queue := make([]string, 0, len(hashes))
	queue = append(queue, hashes...)
	seen := make(map[string]bool)

	for i := 0; i < len(queue); i++ {
		hash := queue[i]
		if seen[hash] {
			continue
		}

		seen[hash] = true

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		ni, err := c.GetNarInfo(ctx, hash)
		if err != nil {
			return nil, err
		}

		missing, err := filter(ctx, ni.StorePath)
		if err != nil {
			return nil, err
		}

		if !missing {
			continue
		}

		fetched[ni.StorePath] = ni

		for _, ref := range ni.References {
			if ref == "" {
				continue
			}

			idx := strings.Index(ref, "-")

			if idx <= 0 {
				continue
			}

			refHash := ref[:idx]
			queue = append(queue, refHash)
		}
	}

	return topoSort(fetched), nil
}

// topoSort returns narinfos in dependency order (leaves first).
func topoSort(infos map[string]*narinfo.NarInfo) []*narinfo.NarInfo {
	deps := make(map[string][]string)

	for path, ni := range infos {
		for _, ref := range ni.References {
			if ref == "" {
				continue
			}

			absRef := storepath.StoreDir + "/" + ref

			if _, ok := infos[absRef]; ok && absRef != path {
				deps[path] = append(deps[path], absRef)
			}
		}
	}

	var result []*narinfo.NarInfo

	visited := make(map[string]bool)

	var visit func(path string)
	visit = func(path string) {
		if visited[path] {
			return
		}

		visited[path] = true

		for _, dep := range deps[path] {
			visit(dep)
		}

		result = append(result, infos[path])
	}

	for path := range infos {
		visit(path)
	}

	return result
}
