package binarycache_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nix-community/go-nix/pkg/binarycache"
	"github.com/nix-community/go-nix/pkg/narinfo"
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

const testNarInfo = `StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz
Compression: xz
FileHash: sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d
FileSize: 114980
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Deriver: 10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv
Sig: cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==
`

func TestGetNarInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/00bgd045z0d4icpbc2yyz4gx48ak44la.narinfo":
			w.Header().Set("Content-Type", "text/x-nix-narinfo")
			w.Write([]byte(testNarInfo))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	ni, err := c.GetNarInfo(context.Background(), "00bgd045z0d4icpbc2yyz4gx48ak44la")
	require.NoError(t, err)
	assert.Equal(t, "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432", ni.StorePath)
	assert.Equal(t, "xz", ni.Compression)
	assert.Equal(t, uint64(464152), ni.NarSize)
	assert.Len(t, ni.References, 1)
}

func TestGetNarInfoNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	_, err := c.GetNarInfo(context.Background(), "00000000000000000000000000000000")
	assert.Error(t, err)
}

func TestGetNar(t *testing.T) {
	narData := []byte("fake-nar-data-for-testing")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/nar/fakehash.nar":
			w.Header().Set("Content-Type", "application/x-nix-archive")
			w.Write(narData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	ni := &narinfo.NarInfo{
		URL:         "nar/fakehash.nar",
		Compression: "none",
	}

	rc, err := c.GetNar(context.Background(), ni)
	require.NoError(t, err)

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	assert.Equal(t, narData, got)
}

func TestResolveClosure(t *testing.T) {
	// A depends on B and C, B depends on C, C has no deps.
	narinfos := map[string]string{
		"/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.narinfo": `StorePath: /nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-a
URL: nar/aaa.nar
Compression: none
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 100
References: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-b cccccccccccccccccccccccccccccccc-c
`,
		"/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.narinfo": `StorePath: /nix/store/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-b
URL: nar/bbb.nar
Compression: none
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 100
References: cccccccccccccccccccccccccccccccc-c
`,
		"/cccccccccccccccccccccccccccccccc.narinfo": `StorePath: /nix/store/cccccccccccccccccccccccccccccccc-c
URL: nar/ccc.nar
Compression: none
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 100
References: 
`,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := narinfos[r.URL.Path]
		if ok {
			w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	allMissing := func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}

	result, err := c.ResolveClosure(context.Background(), []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, allMissing)
	require.NoError(t, err)

	// Should have 3 narinfos, in dependency order (leaves first).
	require.Len(t, result, 3)

	// C should come before B, B before A (leaves first).
	paths := make([]string, len(result))
	for i, ni := range result {
		paths[i] = ni.StorePath
	}
	assert.Equal(t, "/nix/store/cccccccccccccccccccccccccccccccc-c", paths[0])
	assert.Equal(t, "/nix/store/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-b", paths[1])
	assert.Equal(t, "/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-a", paths[2])
}

func TestResolveClosureSkipsPresent(t *testing.T) {
	narinfos := map[string]string{
		"/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.narinfo": `StorePath: /nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-a
URL: nar/aaa.nar
Compression: none
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 100
References: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-b
`,
		"/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.narinfo": `StorePath: /nix/store/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-b
URL: nar/bbb.nar
Compression: none
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 100
References: 
`,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := narinfos[r.URL.Path]
		if ok {
			w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	// B is already present in the store.
	filter := func(_ context.Context, storePath string) (bool, error) {
		return storePath != "/nix/store/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-b", nil
	}

	result, err := c.ResolveClosure(context.Background(), []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, filter)
	require.NoError(t, err)

	// Should only contain A (B is already present).
	require.Len(t, result, 1)
	assert.Equal(t, "/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-a", result[0].StorePath)
}

func TestSubstitute(t *testing.T) {
	narData := []byte("fake-nar-for-substitute")

	narinfos := map[string]string{
		"/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.narinfo": `StorePath: /nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-a
URL: nar/aaa.nar
Compression: none
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 100
References: 
`,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body, ok := narinfos[r.URL.Path]; ok {
			w.Write([]byte(body))
			return
		}
		if r.URL.Path == "/nar/aaa.nar" {
			w.Write(narData)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := binarycache.New(srv.URL)

	allMissing := func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}

	var imported []string
	importer := binarycache.ImporterFunc(func(_ context.Context, info *narinfo.NarInfo, nar io.Reader) error {
		data, err := io.ReadAll(nar)
		if err != nil {
			return err
		}
		assert.Equal(t, narData, data)
		imported = append(imported, info.StorePath)
		return nil
	})

	err := c.Substitute(context.Background(), []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, allMissing, importer)
	require.NoError(t, err)
	assert.Equal(t, []string{"/nix/store/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-a"}, imported)
}
