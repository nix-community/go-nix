package cache

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nix-community/go-nix/pkg/binarycache"
	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/narinfo/signature"
	"github.com/nix-community/go-nix/pkg/nixhash"
	"github.com/nix-community/go-nix/pkg/sqlite"
	"github.com/nix-community/go-nix/pkg/sqlite/binary_cache_v6"
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

	if cmd.Upstream != "" {
		s.upstream = binarycache.New(cmd.Upstream)
		s.cacheDir = cmd.CacheDir

		if err := os.MkdirAll(filepath.Join(cmd.CacheDir, "nar"), 0o755); err != nil {
			return fmt.Errorf("creating cache dir: %w", err)
		}

		db, queries, err := sqlite.BinaryCacheV6(filepath.Join(cmd.CacheDir, "cache.db"))
		if err != nil {
			return fmt.Errorf("opening cache db: %w", err)
		}
		defer db.Close()

		// Create tables if they don't exist (fresh database).
		for _, ddl := range []string{
			`CREATE TABLE IF NOT EXISTS BinaryCaches (
				id        integer primary key autoincrement not null,
				url       text unique not null,
				timestamp integer not null,
				storeDir  text not null,
				wantMassQuery integer not null,
				priority  integer not null
			)`,
			`CREATE TABLE IF NOT EXISTS NARs (
				cache            integer not null,
				hashPart         text not null,
				namePart         text,
				url              text,
				compression      text,
				fileHash         text,
				fileSize         integer,
				narHash          text,
				narSize          integer,
				refs             text,
				deriver          text,
				sigs             text,
				ca               text,
				timestamp        integer not null,
				present          integer not null,
				primary key (cache, hashPart),
				foreign key (cache) references BinaryCaches(id) on delete cascade
			)`,
			`CREATE TABLE IF NOT EXISTS Realisations (
				cache integer not null,
				outputId text not null,
				content blob,
				timestamp        integer not null,
				primary key (cache, outputId),
				foreign key (cache) references BinaryCaches(id) on delete cascade
			)`,
			`CREATE TABLE IF NOT EXISTS LastPurge (
				dummy            text primary key,
				value            integer
			)`,
		} {
			if _, err := db.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("initializing cache schema: %w", err)
			}
		}

		s.cacheDB = queries

		cacheID, err := queries.InsertCache(ctx, binary_cache_v6.InsertCacheParams{
			Url:           cmd.Upstream,
			Timestamp:     time.Now().Unix(),
			Storedir:      storepath.StoreDir,
			Wantmassquery: 1,
			Priority:      int64(cmd.Priority),
		})
		if err != nil {
			return fmt.Errorf("registering cache: %w", err)
		}
		s.cacheID = cacheID

		log.Printf("proxy mode: upstream %s, cache dir %s", cmd.Upstream, cmd.CacheDir)
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
	// Proxy fields (nil when no upstream configured).
	upstream *binarycache.Client
	cacheDB  *binary_cache_v6.Queries
	cacheID  int64
	cacheDir string
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
		if s.upstream != nil {
			s.handleProxyNarInfo(w, r, hash)
			return
		}
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
	if len(parts) != 2 || !isValidNixBase32(parts[0]) || !isValidNixBase32(parts[1]) {
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
		if s.upstream != nil {
			s.handleProxyNar(w, r, outhash, parts[1])
			return
		}
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

func (s *server) handleProxyNarInfo(w http.ResponseWriter, r *http.Request, hash string) {
	// Check SQLite cache first.
	if ni, ok := s.cachedNarInfo(r.Context(), hash); ok {
		signNarInfo(ni, s.signKeys)
		w.Header().Set("Content-Type", ni.ContentType())
		fmt.Fprint(w, ni.String())
		return
	}

	ni, err := s.upstream.GetNarInfo(r.Context(), hash)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Rewrite URL to use outhash-narhash pattern.
	// GetNar decompresses, so we serve uncompressed NARs.
	outhash := extractHash(ni.StorePath)
	narhash := ni.NarHash.Format(nixhash.NixBase32, false)
	ni.URL = fmt.Sprintf("nar/%s-%s.nar", outhash, narhash)
	ni.Compression = "none"
	ni.FileHash = ni.NarHash
	ni.FileSize = ni.NarSize

	// Cache narinfo in SQLite.
	s.cacheNarInfo(r.Context(), hash, ni)

	signNarInfo(ni, s.signKeys)

	w.Header().Set("Content-Type", ni.ContentType())
	fmt.Fprint(w, ni.String())
}

func (s *server) cacheNarInfo(ctx context.Context, hash string, ni *narinfo.NarInfo) {
	if s.cacheDB == nil {
		return
	}

	var sigs []string
	for _, sig := range ni.Signatures {
		sigs = append(sigs, sig.String())
	}

	if err := s.cacheDB.InsertNar(ctx, binary_cache_v6.InsertNarParams{
		Cache:       s.cacheID,
		Hashpart:    hash,
		Namepart:    toNullString(strings.TrimPrefix(ni.StorePath, storepath.StoreDir+"/")),
		Url:         toNullString(ni.URL),
		Compression: toNullString(ni.Compression),
		Filehash:    hashToNullString(ni.FileHash),
		Filesize:    sql.NullInt64{Int64: int64(ni.FileSize), Valid: true},
		Narhash:     toNullString(ni.NarHash.String()),
		Narsize:     sql.NullInt64{Int64: int64(ni.NarSize), Valid: true},
		Refs:        toNullString(strings.Join(ni.References, " ")),
		Deriver:     toNullString(ni.Deriver),
		Sigs:        toNullString(strings.Join(sigs, " ")),
		Ca:          toNullString(ni.CA),
		Timestamp:   time.Now().Unix(),
	}); err != nil {
		log.Printf("caching narinfo for %s: %v", hash, err)
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func hashToNullString(h *nixhash.HashWithEncoding) sql.NullString {
	if h == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: h.String(), Valid: true}
}

func isValidNixBase32(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'd') ||
			(c >= 'f' && c <= 'n') || (c >= 'p' && c <= 's') ||
			(c >= 'v' && c <= 'z')) {
			return false
		}
	}
	return true
}

func (s *server) cachedNarInfo(ctx context.Context, hash string) (*narinfo.NarInfo, bool) {
	if s.cacheDB == nil {
		return nil, false
	}

	rows, err := s.cacheDB.QueryNar(ctx, binary_cache_v6.QueryNarParams{
		Cache:       s.cacheID,
		Hashpart:    hash,
		Timestamp:   0, // negative TTL: never expire absent entries
		Timestamp_2: 0, // positive TTL: never expire present entries
	})
	if err != nil || len(rows) == 0 {
		return nil, false
	}
	row := rows[0]
	if row.Present == 0 {
		return nil, false
	}

	narHash, err := nixhash.ParseAny(row.Narhash.String, nil)
	if err != nil {
		return nil, false
	}

	ni := &narinfo.NarInfo{
		StorePath:   storepath.StoreDir + "/" + row.Namepart.String,
		URL:         row.Url.String,
		Compression: row.Compression.String,
		NarHash:     narHash,
		NarSize:     uint64(row.Narsize.Int64),
		CA:          row.Ca.String,
	}

	if row.Filehash.Valid {
		fh, err := nixhash.ParseAny(row.Filehash.String, nil)
		if err == nil {
			ni.FileHash = fh
		}
	}
	if row.Filesize.Valid {
		ni.FileSize = uint64(row.Filesize.Int64)
	}

	if row.Refs.Valid && row.Refs.String != "" {
		ni.References = strings.Split(row.Refs.String, " ")
	}
	if row.Deriver.Valid {
		ni.Deriver = row.Deriver.String
	}
	if row.Sigs.Valid && row.Sigs.String != "" {
		for _, s := range strings.Split(row.Sigs.String, " ") {
			sig, err := signature.ParseSignature(s)
			if err != nil {
				continue
			}
			ni.Signatures = append(ni.Signatures, sig)
		}
	}

	return ni, true
}

func (s *server) handleProxyNar(w http.ResponseWriter, r *http.Request, outhash, narhash string) {
	narFile := filepath.Join(s.cacheDir, "nar", narhash+".nar")

	// Serve from disk cache if available.
	if _, err := os.Stat(narFile); err == nil {
		w.Header().Set("Content-Type", "application/x-nix-archive")
		http.ServeFile(w, r, narFile)
		return
	}

	// Fetch narinfo to get the upstream NAR URL.
	ni, err := s.upstream.GetNarInfo(r.Context(), outhash)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	rc, err := s.upstream.GetNar(r.Context(), ni)
	if err != nil {
		log.Printf("GetNar(%s): %v", outhash, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	// Tee to disk and response.
	tmp, err := os.CreateTemp(filepath.Join(s.cacheDir, "nar"), ".tmp-*")
	if err != nil {
		log.Printf("creating temp file: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	tmpPath := tmp.Name()

	w.Header().Set("Content-Type", "application/x-nix-archive")
	mw := io.MultiWriter(w, tmp)
	_, copyErr := io.Copy(mw, rc)
	tmp.Close()

	if copyErr != nil {
		log.Printf("streaming proxy NAR for %s: %v", outhash, copyErr)
		os.Remove(tmpPath)
		return
	}

	// Atomically move into place.
	if err := os.Rename(tmpPath, narFile); err != nil {
		log.Printf("caching NAR %s: %v", narhash, err)
		os.Remove(tmpPath)
	}
}
