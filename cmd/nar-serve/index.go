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
	"github.com/zimbatm/go-nix/src/libstore"
	"github.com/zimbatm/go-nix/src/nar"
)

// TODO: make that configurable
var nixCache = libstore.DefaultCache

const indexPage = `
<pre>
#     #     #     ######             #####   #######  ######   #     #  #######  
##    #    # #    #     #           #     #  #        #     #  #     #  #        
# #   #   #   #   #     #           #        #        #     #  #     #  #        
#  #  #  #     #  ######   #######   #####   #####    ######   #     #  #####    
#   # #  #######  #   #                   #  #        #   #     #   #   #        
#    ##  #     #  #    #            #     #  #        #    #     # #    #        
#     #  #     #  #     #            #####   #######  #     #     #     #######  

Unpack and serve the content of NAR files straight from [[ https://cache.nixos.org ]]

Pick a NAR path on your filesystem and paste it at the end of the URL.


Examples:

  * <a href="/nix/store/zk5crljigizl5snkfyaijja89bb6228x-rake-12.3.1/bin/rake">readlink -f $(which rake)</a>
  * <a href="/nix/store/barxv95b8arrlh97s6axj8k7ljn7aky1-go-1.12/share/go/doc/effective_go.html">/nix/store/barxv95b8arrlh97s6axj8k7ljn7aky1-go-1.12/share/go/doc/effective_go.html</a>

Like this project? Star it on <a href="https://github.com/zimbatm/go-nix">GitHub</a>.
`

// TODO: consider keeping a LRU cache
func getNarInfo(key string) (*libstore.NarInfo, error) {
	path := fmt.Sprintf("%s.narinfo", key)
	fmt.Println("Fetching the narinfo:", path)
	r, err := nixCache.GetFile(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return libstore.ParseNarInfo(r)
}


// Handler is the entry-point for @now/go as well as the stub main.go net/http
func Handler(w http.ResponseWriter, req *http.Request) {
	path := strings.Trim(req.URL.Path, "/")
	path = strings.TrimPrefix(path, "nix/store/") // allow to paste from the filesystem
	components := strings.Split(path, "/")
	if len(components) < 2 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(indexPage))
		return
	}
	fmt.Println(len(components), components)

	narName := strings.Split(components[0], "-")[0]

	// Get the NAR info to find the NAR
	narinfo, err := getNarInfo(narName)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Println("narinfo", narinfo)

	// TODO: consider keeping a LRU cache
	narPATH := narinfo.URL
	fmt.Println("fetching the NAR:", narPATH)
	file, err := nixCache.GetFile(narPATH)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer file.Close()

	var r io.Reader
	r = file

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
	newPath := strings.Join(components[1:], "/")

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
