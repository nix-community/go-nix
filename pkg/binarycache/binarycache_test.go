package binarycache_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nix-community/go-nix/pkg/binarycache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCacheInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/nix-cache-info" {
			w.Write([]byte("StoreDir: /nix/store\nWantMassQuery: 1\nPriority: 40\n"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	info, err := c.GetCacheInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/nix/store", info.StoreDir)
	assert.True(t, info.WantMassQuery)
	assert.Equal(t, 40, info.Priority)
}

func TestGetCacheInfoDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/nix-cache-info" {
			w.Write([]byte("StoreDir: /nix/store\n"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	info, err := c.GetCacheInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/nix/store", info.StoreDir)
	assert.False(t, info.WantMassQuery)
	assert.Equal(t, 0, info.Priority)
}

func TestGetCacheInfoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	_, err := c.GetCacheInfo(context.Background())
	assert.Error(t, err)
}
