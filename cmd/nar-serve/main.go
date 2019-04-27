package main

import (
	"compress/bzip2"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
	"github.com/urfave/negroni"
	"github.com/zimbatm/go-nix/src/libstore"
	"github.com/zimbatm/go-nix/src/nar"
)

const nixCache = "https://cache.nixos.org"

// TODO: make upstream configurable
// TODO: consider keeping a LRU cache
func getNarInfo(key string) (*libstore.NarInfo, error) {
	url := fmt.Sprintf("%s/%s.narinfo", nixCache, key)
	fmt.Println("Fetching the narinfo:", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("expected status 200, got %s", resp.Status)
	}
	return libstore.ParseNarInfo(resp.Body)
}

func serveNAR(w http.ResponseWriter, req *http.Request) {
	components := strings.Split(req.URL.Path, "/")
	if len(components) <= 1 {
		// TODO: serve index page
		http.Error(w, "need a NAR path", 400)
		return
	}
	if components[0] != "" {
		http.Error(w, "expected first component to be empty", 500)
		return
	}

	// Get the NAR info to find the NAR
	narinfo, err := getNarInfo(components[1])
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Println("narinfo", narinfo)

	// TODO: consider keeping a LRU cache
	narURL := fmt.Sprintf("%s/%s", nixCache, narinfo.URL)
	fmt.Println("fetching the NAR:", narURL)
	resp, err := http.Get(narURL)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var r io.Reader
	r = resp.Body

	// decompress on the fly
	switch narinfo.Compression {
	case "xz":
		r, err = xz.NewReader(r)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	case "bzip2":
		r = bzip2.NewReader(r)
	default:
		http.Error(w, fmt.Sprintf("compression %s not handled", narinfo.Compression), 500)
		return
	}

	narReader := nar.NewReader(r)
	newPath := strings.Join(components[2:], "/")

	fmt.Println("newPath", newPath)

	for {
		hdr, err := narReader.Next()
		if err != nil {
			if err == io.EOF {
				http.Error(w, "file not found", 404)
			} else {
				http.Error(w, err.Error(), 500)
			}
			return
		}

		// we've got a match!
		if hdr.Name == newPath {
			switch hdr.Type {
			case nar.TypeDirectory:
				fmt.Fprintf(w, "found a directory here")
			case nar.TypeSymlink:
				fmt.Fprintf(w, "found a symlink to %s", hdr.Linkname)
			case nar.TypeRegular:
				// TODO: ETag header matching. Use the NAR file name as the ETag
				// TODO: expose the executable flag somehow?
				ctype := mime.TypeByExtension(filepath.Ext(hdr.Name))
				if ctype == "" {
					ctype = "application/octet-stream"
					// TODO: use http.DetectContentType as a fallback
				}

				w.Header().Set("Cache-Control", "immutable")
				w.Header().Set("Content-Type", ctype)
				w.Header().Set("Content-Length", fmt.Sprintf("%d", hdr.Size))
				if req.Method != "HEAD" {
					io.CopyN(w, narReader, hdr.Size)
				}
			default:
				http.Error(w, fmt.Sprintf("BUG: unknown NAR header type: %s", hdr.Type), 500)
			}
			return
		}

		// TODO: since the nar entries are sorted it's possible to abort early by
		//       comparing the paths
	}
}

func main() {
	n := negroni.Classic() // Includes some default middlewares
	n.UseHandler(http.HandlerFunc(serveNAR))

	addr := ":3000"
	fmt.Println("Starting server on address", addr)
	err := http.ListenAndServe(addr, n)
	if err != nil {
		panic(err)
	}
}
