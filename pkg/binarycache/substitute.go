package binarycache

import (
	"context"
	"fmt"
	"io"

	"github.com/nix-community/go-nix/pkg/narinfo"
)

// ImporterFunc is an adapter to allow the use of ordinary functions as Importers.
type ImporterFunc func(ctx context.Context, info *narinfo.NarInfo, nar io.Reader) error

func (f ImporterFunc) Import(ctx context.Context, info *narinfo.NarInfo, nar io.Reader) error {
	return f(ctx, info, nar)
}

// Substitute fetches and imports store paths and all their missing dependencies
// from the binary cache. It resolves the closure, downloads NARs, and feeds
// them to the Importer in dependency order (leaves first).
func (c *Client) Substitute(
	ctx context.Context,
	hashes []string,
	filter PathFilter,
	importer Importer,
) error {
	closure, err := c.ResolveClosure(ctx, hashes, filter)
	if err != nil {
		return fmt.Errorf("resolve closure: %w", err)
	}

	for _, ni := range closure {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := c.substituteOne(ctx, ni, importer); err != nil {
			return fmt.Errorf("substitute %s: %w", ni.StorePath, err)
		}
	}

	return nil
}

func (c *Client) substituteOne(ctx context.Context, ni *narinfo.NarInfo, importer Importer) error {
	rc, err := c.GetNar(ctx, ni)
	if err != nil {
		return err
	}
	defer rc.Close()

	return importer.Import(ctx, ni, rc)
}
