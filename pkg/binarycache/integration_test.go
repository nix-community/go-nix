//go:build integration

package binarycache_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/binarycache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationGetCacheInfo(t *testing.T) {
	c := binarycache.New("https://cache.nixos.org")

	info, err := c.GetCacheInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/nix/store", info.StoreDir)
	assert.True(t, info.WantMassQuery)
	t.Logf("Priority: %d", info.Priority)
}

func TestIntegrationGetNarInfo(t *testing.T) {
	c := binarycache.New("https://cache.nixos.org")

	ni, err := c.GetNarInfo(context.Background(), "00bgd045z0d4icpbc2yyz4gx48ak44la")
	if err != nil {
		t.Skipf("narinfo not found (may have been GC'd from cache): %v", err)
	}

	assert.Contains(t, ni.StorePath, "/nix/store/")
	assert.NotEmpty(t, ni.URL)
	assert.True(t, ni.NarSize > 0)
	t.Logf("StorePath: %s, NarSize: %d, Compression: %s", ni.StorePath, ni.NarSize, ni.Compression)
}

func TestIntegrationGetNar(t *testing.T) {
	c := binarycache.New("https://cache.nixos.org")

	ni, err := c.GetNarInfo(context.Background(), "00bgd045z0d4icpbc2yyz4gx48ak44la")
	if err != nil {
		t.Skipf("narinfo not found: %v", err)
	}

	rc, err := c.GetNar(context.Background(), ni)
	require.NoError(t, err)
	defer rc.Close()

	// Read first 64 bytes to check for NAR magic.
	buf := make([]byte, 64)
	n, err := io.ReadAtLeast(rc, buf, 13)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(buf[:n]), "nix-archive-1"),
		"NAR data should contain nix-archive-1 magic")

	t.Logf("Successfully fetched and decompressed NAR (%s)", ni.Compression)
}

func TestIntegrationResolveClosure(t *testing.T) {
	c := binarycache.New("https://cache.nixos.org")

	allMissing := func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}

	result, err := c.ResolveClosure(context.Background(), []string{"00bgd045z0d4icpbc2yyz4gx48ak44la"}, allMissing)
	if err != nil {
		t.Skipf("could not resolve closure: %v", err)
	}

	require.True(t, len(result) >= 1, "closure should have at least the path itself")

	for _, ni := range result {
		t.Logf("  %s (%d bytes)", ni.StorePath, ni.NarSize)
	}
}
