package cache

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/narinfo/signature"
	"github.com/nix-community/go-nix/pkg/nixhash"
	"github.com/nix-community/go-nix/pkg/storepath"
)

// ServeCmd serves the local Nix store as an HTTP binary cache.
type ServeCmd struct {
	Bind     string   `kong:"optional,default=':5000',help='Listen address'"`
	Socket   string   `kong:"optional,default='/nix/var/nix/daemon-socket/socket',help='Nix daemon socket path'"`
	SignKeys []string `kong:"optional,name='sign-key',help='Secret key file paths for signing narinfo'"`
	Upstream string   `kong:"optional,help='Upstream binary cache URL (enables proxy mode)'"`
	CacheDir string   `kong:"optional,default='./cache',help='Disk cache directory for proxy mode'"`
	Priority int      `kong:"optional,default='30',help='Cache priority for /nix-cache-info'"`
}

func (cmd *ServeCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load signing keys.
	var signKeys []signature.SecretKey
	for _, path := range cmd.SignKeys {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading sign key %s: %w", path, err)
		}
		sk, err := signature.LoadSecretKey(strings.TrimSpace(string(data)))
		if err != nil {
			return fmt.Errorf("parsing sign key %s: %w", path, err)
		}
		signKeys = append(signKeys, sk)
	}

	// Connect to daemon.
	client, err := daemon.Connect(cmd.Socket)
	if err != nil {
		return fmt.Errorf("connect to daemon: %w", err)
	}
	defer client.Close()

	s := &server{
		daemon:   client,
		signKeys: signKeys,
		priority: cmd.Priority,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/nix-cache-info", s.handleCacheInfo)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/{hash}.narinfo", s.handleNarInfo)
	mux.HandleFunc("/nar/{narpath}", s.handleNar)

	srv := &http.Server{Addr: cmd.Bind, Handler: mux}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Printf("serving on %s", cmd.Bind)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

type server struct {
	daemon   *daemon.Client
	signKeys []signature.SecretKey
	priority int
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func (s *server) handleCacheInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-nix-cache-info")
	fmt.Fprintf(w, "StoreDir: %s\n", storepath.StoreDir)
	fmt.Fprintf(w, "WantMassQuery: 1\n")
	fmt.Fprintf(w, "Priority: %d\n", s.priority)
}

func (s *server) handleNarInfo(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if hash == "" {
		http.NotFound(w, r)
		return
	}

	sp, err := s.daemon.QueryPathFromHashPart(r.Context(), hash)
	if err != nil {
		log.Printf("QueryPathFromHashPart(%s): %v", hash, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if sp == "" {
		http.NotFound(w, r)
		return
	}

	info, err := s.daemon.QueryPathInfo(r.Context(), sp)
	if err != nil {
		log.Printf("QueryPathInfo(%s): %v", sp, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	ni, err := pathInfoToNarInfo(info)
	if err != nil {
		log.Printf("pathInfoToNarInfo(%s): %v", sp, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	signNarInfo(ni, s.signKeys)

	w.Header().Set("Content-Type", ni.ContentType())
	fmt.Fprint(w, ni.String())
}

func pathInfoToNarInfo(info *daemon.PathInfo) (*narinfo.NarInfo, error) {
	narHash, err := nixhash.ParseAny(info.NarHash, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing NarHash: %w", err)
	}

	ni := &narinfo.NarInfo{
		StorePath:   info.StorePath,
		URL:         fmt.Sprintf("nar/%s-%s.nar", extractHash(info.StorePath), narHash.Format(nixhash.NixBase32, false)),
		Compression: "none",
		FileHash:    narHash,
		FileSize:    info.NarSize,
		NarHash:     narHash,
		NarSize:     info.NarSize,
		CA:          info.CA,
	}

	// Convert absolute references to relative.
	for _, ref := range info.References {
		ni.References = append(ni.References, strings.TrimPrefix(ref, storepath.StoreDir+"/"))
	}

	// Convert absolute deriver to relative.
	if info.Deriver != "" {
		ni.Deriver = strings.TrimPrefix(info.Deriver, storepath.StoreDir+"/")
	}

	// Parse existing signatures.
	for _, s := range info.Sigs {
		sig, err := signature.ParseSignature(s)
		if err != nil {
			continue
		}
		ni.Signatures = append(ni.Signatures, sig)
	}

	return ni, nil
}

func signNarInfo(ni *narinfo.NarInfo, keys []signature.SecretKey) {
	if len(keys) == 0 {
		return
	}
	fp := ni.Fingerprint()
	for _, sk := range keys {
		sig, err := sk.Sign(nil, fp)
		if err != nil {
			continue
		}
		ni.Signatures = append(ni.Signatures, sig)
	}
}

func (s *server) handleNar(w http.ResponseWriter, r *http.Request) {
	narpath := r.PathValue("narpath")

	// Parse {outhash}-{narhash}.nar
	narpath = strings.TrimSuffix(narpath, ".nar")
	parts := strings.SplitN(narpath, "-", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.NotFound(w, r)
		return
	}
	outhash := parts[0]

	sp, err := s.daemon.QueryPathFromHashPart(r.Context(), outhash)
	if err != nil {
		log.Printf("QueryPathFromHashPart(%s): %v", outhash, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if sp == "" {
		http.NotFound(w, r)
		return
	}

	rc, err := s.daemon.NarFromPath(r.Context(), sp)
	if err != nil {
		log.Printf("NarFromPath(%s): %v", sp, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/x-nix-archive")
	if _, err := io.Copy(w, rc); err != nil {
		log.Printf("streaming NAR for %s: %v", sp, err)
	}
}
