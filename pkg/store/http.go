package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sync"

	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/nix-community/go-nix/pkg/nar/narinfo"
	"github.com/nix-community/go-nix/pkg/nixbase32"
	"github.com/nix-community/go-nix/pkg/nixpath"
)

var _ Store = &HTTPStore{}

// HTTPStore exposes a HTTP binary cache using .narinfo and .nar files
// It uses/populates another Store and ChunkStore as a cache,
// The ChunkStore also needs to be populated with the chunks
// referred in the PathInfo of a Put() call.
// TODO: Allow configuring to validate signatures.
type HTTPStore struct {
	cacheStore      Store
	cacheChunkStore ChunkStore
	Client          *http.Client
	BaseURL         *url.URL
}

// getNarinfoURL returns the full URL to a .narinfo,
// with respect to the configured baseURL.
// It constructs the URL by extracting the hash from the outputPath URL.
func (hs *HTTPStore) getNarinfoURL(outputPath *nixpath.NixPath) url.URL {
	// copy the base URL
	url := *hs.BaseURL

	url.Path = path.Join(url.Path, nixbase32.EncodeToString(outputPath.Digest)+".narinfo")

	return url
}

// getNarURL returns the full URL to a .nar file,
// with respect to the configured baseURL
// It constructs the full URL by appending the passed URL to baseURL
// The passed URL usually comes from the `URL` field in the .narinfo file.
func (hs *HTTPStore) getNarURL(narPath string) url.URL {
	// copy the base URL
	url := *hs.BaseURL

	url.Path = path.Join(url.Path, narPath)

	return url
}

func (hs *HTTPStore) Get(ctx context.Context, outputPath string) (*PathInfo, error) {
	np, err := nixpath.FromString(outputPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing output path: %w", err)
	}

	// check the underlying cache store first
	has, err := hs.cacheStore.Has(ctx, outputPath)
	if err != nil {
		return nil, fmt.Errorf("error asking underlying cache store: %w", err)
	}
	// if it's in there, we can just pass it along
	if has {
		return hs.cacheStore.Get(ctx, outputPath)
	}

	// else, cause substitution.
	niURL := hs.getNarinfoURL(np)

	// construct the request
	niReq, err := http.NewRequestWithContext(ctx, "GET", niURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error constructing request: %w", err)
	}

	// do the request for the .narinfo file
	niResp, err := hs.Client.Do(niReq)
	if err != nil {
		return nil, fmt.Errorf("error doing narinfo request: %w", err)
	}
	defer niResp.Body.Close()

	if niResp.StatusCode < 200 || niResp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status code retrieving narinfo: %v", niResp.StatusCode)
	}

	ni, err := narinfo.Parse(niResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing narinfo: %w", err)
	}

	// ensure the .narinfo mentions the same store path that we expected
	if ni.StorePath != outputPath {
		return nil, fmt.Errorf("narinfo shows wrong storepath, got %v, expected %v", ni.StorePath, outputPath)
	}

	// some more basic consistency checks of the .narinfo
	if err := ni.Check(); err != nil {
		return nil, fmt.Errorf(".narinfo fails consistency check: %w", err)
	}

	// TODO: signature checks

	// construct the URL for the .nar file
	narURL := hs.getNarURL(ni.URL)

	// construct the request for the .nar file
	narReq, err := http.NewRequestWithContext(ctx, "GET", narURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error constructing nar request: %w", err)
	}

	// do the request for the .nar file
	narResp, err := hs.Client.Do(narReq)
	if err != nil {
		return nil, fmt.Errorf("error doing nar request: %w", err)
	}
	defer narResp.Body.Close()

	if niResp.StatusCode < 200 || niResp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status code retrieving nar: %v", niResp.StatusCode)
	}

	// TODO: handle compression.
	// In case of compression AND provided FilePath/FileSize, also check these?

	hashWriter, err := hash.NewWriter(ni.NarHash.HashType)
	if err != nil {
		return nil, fmt.Errorf("error constructing hash.Writer: %w", err)
	}

	// setup a io.TeeReader to ensure the nar file contents get written to hashWriter
	// while the NarReader reads through the body.
	tr := io.TeeReader(narReq.Body, hashWriter)

	// receive the .nar file
	pathInfo, err := Import(ctx, ni.StorePath, ni.References, tr, hs.cacheChunkStore)
	if err != nil {
		return nil, fmt.Errorf("error converting narinfo and nar to pathInfo: %w", err)
	}

	// query the hashWriter if size matches.
	if ni.NarSize != hashWriter.BytesWritten() {
		return nil, fmt.Errorf("read %v bytes of nar, expected %v", ni.NarSize, hashWriter.BytesWritten())
	}

	// query the hashWriter if hash matches
	if !bytes.Equal(ni.NarHash.Digest, hashWriter.Digest()) {
		return nil, fmt.Errorf("got %s:%s as NarHash, expected %s",
			ni.NarHash.HashType,
			nixbase32.EncodeToString(hashWriter.Digest()),
			ni.NarHash,
		)
	}

	return pathInfo, nil
}

func (hs *HTTPStore) Has(ctx context.Context, outputPath string) (bool, error) {
	np, err := nixpath.FromString(outputPath)
	if err != nil {
		return false, fmt.Errorf("error parsing output path: %w", err)
	}

	// check the underlying cache store first
	has, err := hs.cacheStore.Has(ctx, outputPath)
	if err != nil {
		return false, fmt.Errorf("error asking underlying cache store: %w", err)
	}
	// if it's in there, we can return true.
	if has {
		return true, nil
	}

	// else, we peek at the .narinfo file with a HEAD request.
	niURL := hs.getNarinfoURL(np)

	// construct the request
	niReq, err := http.NewRequestWithContext(ctx, "HEAD", niURL.String(), nil)
	if err != nil {
		return false, fmt.Errorf("error constructing request: %w", err)
	}

	// do the request for the .narinfo file
	niResp, err := hs.Client.Do(niReq)
	if err != nil {
		return false, fmt.Errorf("error doing narinfo request: %w", err)
	}
	defer niResp.Body.Close()

	// if we get a 404, we assume it doesn't exist.
	if niResp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// return an error in case other errors occur
	// TODO: check if there's other status codes that should return false, nil
	if niResp.StatusCode < 200 || niResp.StatusCode >= 300 {
		return false, fmt.Errorf("bad status code retrieving narinfo: %v", niResp.StatusCode)
	}

	// else, return true.
	return true, nil
}

func (hs *HTTPStore) Put(ctx context.Context, pathInfo *PathInfo) error {
	np, err := nixpath.FromString(pathInfo.OutputName)
	if err != nil {
		return fmt.Errorf("error parsing output path: %w", err)
	}

	// Usually NAR files are uploaded to /nar/$narhash.nar[.$compressionSuffix].
	// However, this means we need to know the NARHash before uploading the files, as
	// a plain HTTP PUT call contains the destination path, and we can't move files after upload.

	// This means, we render the NAR file twice - once to calculate NarHash, NarSize,
	// a second time to do the actual upload.

	// create a hashWriter to calculate NARHash and NARSize.
	// TODO: make hash function configurable?
	hashWriter, err := hash.NewWriter(hash.HashTypeSha512)
	if err != nil {
		return fmt.Errorf("error constructing hash.Writer: %w", err)
	}

	// This costs a bit more CPU, but is better than keeping the (potentially large) NAR file in memory.
	_, _, err = Export(ctx, pathInfo, hashWriter, hs.cacheChunkStore)
	if err != nil {
		return fmt.Errorf("failed to export NAR to hashwriter: %w", err)
	}

	narSize := hashWriter.BytesWritten()
	narHash := hash.Hash{
		HashType: hash.HashTypeSha512,
		Digest:   hashWriter.Digest(),
	}

	// determine the nar url. use $narhash.nar.
	// TODO: once compression is supported, use compression suffix too
	narURLRel := "nar/" + nixbase32.EncodeToString(narHash.Digest) + ".nar"
	narURL := hs.getNarURL(narURLRel)

	// set up the io.Pipe, and an upload context.
	// Have Export produce a NAR file, and provide the pipe reader side to the http request.
	// In case of an error during NAR rendering, cancel the upload.
	pipeReader, pipeWriter := io.Pipe()

	narCtx, cancelNar := context.WithCancel(ctx)

	// construct the request to upload the .nar file
	narReq, err := http.NewRequestWithContext(narCtx, "PUT", narURL.String(), pipeReader)

	// create a buffered narErrors channel.
	narErrors := make(chan error, 2)

	// create a wait group. We use it to ensure both channels exited.
	// TODO: can we replace the wait group, and use the context cancellation for synchronization?
	var wg sync.WaitGroup

	wg.Add(2)

	// spin up a goroutine for the export to NAR
	go func() {
		defer wg.Done()
		defer pipeWriter.Close()

		// Export the NAR, write to the pipe writer
		_, _, err = Export(narCtx, pathInfo, pipeWriter, hs.cacheChunkStore)
		if err != nil {
			narErrors <- fmt.Errorf("failed to export NAR to hashwriter: %w", err)

			cancelNar()
		}
	}()

	// spin up a goroutine for the HTTP upload
	go func() {
		defer wg.Done()
		defer pipeReader.Close()

		// upload the NAR file
		narResp, err := hs.Client.Do(narReq)
		if err != nil {
			narErrors <- fmt.Errorf("error doing nar request: %w", err)

			cancelNar()
		}

		defer narResp.Body.Close()

		if narResp.StatusCode < 200 || narResp.StatusCode >= 300 {
			narErrors <- fmt.Errorf("bad status code retrieving nar: %v", narResp.StatusCode)

			cancelNar()
		}
	}()

	wg.Wait()

	for err := range narErrors {
		return err
	}

	ni := narinfo.NarInfo{
		StorePath: pathInfo.OutputName,
		URL:       narURLRel,

		Compression: "none", // TODO
		FileHash:    &narHash,
		FileSize:    narSize,

		NarHash: &narHash,
		NarSize: narSize,

		References: pathInfo.References,
		// Deriver: "", // TODO
		// System: "", // TODO
		// Signatures: , // TODO
		// CA: , // TODO
	}

	niURL := hs.getNarinfoURL(np)

	// construct the request to upload the .narinfo file
	niReq, err := http.NewRequestWithContext(ctx, "PUT", niURL.String(), bytes.NewBufferString(ni.String()))
	if err != nil {
		return fmt.Errorf("error constructing request: %w", err)
	}

	// do the request to upload the .narinfo file
	niResp, err := hs.Client.Do(niReq)
	if err != nil {
		return fmt.Errorf("error uploading narinfo: %w", err)
	}
	defer niResp.Body.Close()

	// if we get a non-200-y status code, expect the upload to have failed.
	if niResp.StatusCode < 200 || niResp.StatusCode >= 300 {
		return fmt.Errorf("bad status code uploading narinfo: %v", niResp.StatusCode)
	}

	// finally, insert it into the cacheStore
	if err := hs.cacheStore.Put(ctx, pathInfo); err != nil {
		return fmt.Errorf("error putting pathinfo into cache store: %w", err)
	}

	return nil
}
