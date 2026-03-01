package binarycache

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/narinfo/signature"
)

// CacheInfo holds the response from /nix-cache-info.
type CacheInfo struct {
	StoreDir      string
	WantMassQuery bool
	Priority      int
}

// PathFilter decides whether a store path needs to be fetched.
// Returns true if the path is missing and should be downloaded.
type PathFilter func(ctx context.Context, storePath string) (bool, error)

// Importer receives a NAR stream and imports it into the store.
type Importer interface {
	Import(ctx context.Context, info *narinfo.NarInfo, nar io.Reader) error
}

// Client fetches store paths from a Nix binary cache over HTTP.
type Client struct {
	baseURL    string
	httpClient *http.Client
	publicKeys []signature.PublicKey

	infoOnce sync.Once
	info     *CacheInfo
	infoErr  error
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) { cl.httpClient = c }
}

// WithPublicKeys sets public keys for narinfo signature verification.
func WithPublicKeys(keys []signature.PublicKey) Option {
	return func(cl *Client) { cl.publicKeys = keys }
}

// New creates a binary cache client for the given base URL.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}
