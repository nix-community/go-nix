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

	infoMu sync.Mutex
	info   *CacheInfo
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
// The result is cached after the first successful call. Transient errors
// are not cached — a subsequent call will retry the fetch.
func (c *Client) GetCacheInfo(ctx context.Context) (*CacheInfo, error) {
	c.infoMu.Lock()
	defer c.infoMu.Unlock()

	if c.info != nil {
		return c.info, nil
	}

	info, err := c.fetchCacheInfo(ctx)
	if err != nil {
		return nil, err
	}

	c.info = info
	return c.info, nil
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

// GetNar downloads and decompresses a NAR archive. The returned ReadCloser
// streams the uncompressed NAR data. The caller must close it when done.
func (c *Client) GetNar(ctx context.Context, ni *narinfo.NarInfo) (io.ReadCloser, error) {
	url := c.baseURL + "/" + ni.URL

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", ni.URL, resp.Status)
	}

	dr, err := decompress(resp.Body, ni.Compression)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}

	return &narReadCloser{decompressed: dr, body: resp.Body}, nil
}

type narReadCloser struct {
	decompressed io.ReadCloser
	body         io.ReadCloser
}

func (n *narReadCloser) Read(p []byte) (int, error) {
	return n.decompressed.Read(p)
}

func (n *narReadCloser) Close() error {
	err1 := n.decompressed.Close()
	err2 := n.body.Close()

	if err1 != nil {
		return err1
	}

	return err2
}
