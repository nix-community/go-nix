package binarycache

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// GetCacheInfo fetches and parses /nix-cache-info from the binary cache.
// The result is cached after the first successful call.
func (c *Client) GetCacheInfo(ctx context.Context) (*CacheInfo, error) {
	c.infoOnce.Do(func() {
		c.info, c.infoErr = c.fetchCacheInfo(ctx)
	})

	return c.info, c.infoErr
}

func (c *Client) fetchCacheInfo(ctx context.Context) (*CacheInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/nix-cache-info", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /nix-cache-info: %s", resp.Status)
	}

	info := &CacheInfo{}
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}

		switch k {
		case "StoreDir":
			info.StoreDir = v
		case "WantMassQuery":
			info.WantMassQuery = v == "1"
		case "Priority":
			info.Priority, _ = strconv.Atoi(v)
		}
	}

	return info, scanner.Err()
}

// GetNarInfo fetches and parses the .narinfo for a store path hash.
// The hash is the 32-char nixbase32 prefix from the store path.
// If public keys are configured, the narinfo signature is verified.
func (c *Client) GetNarInfo(ctx context.Context, hash string) (*narinfo.NarInfo, error) {
	url := c.baseURL + "/" + hash + ".narinfo"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s.narinfo: %s", hash, resp.Status)
	}

	ni, err := narinfo.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse %s.narinfo: %w", hash, err)
	}

	if err := ni.Check(); err != nil {
		return nil, fmt.Errorf("check %s.narinfo: %w", hash, err)
	}

	if len(c.publicKeys) > 0 {
		if !signature.VerifyFirst(ni.Fingerprint(), ni.Signatures, c.publicKeys) {
			return nil, fmt.Errorf("signature verification failed for %s", hash)
		}
	}

	return ni, nil
}
