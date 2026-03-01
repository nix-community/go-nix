package cache

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/nix-community/go-nix/pkg/binarycache"
	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/storepath"
)

// Cmd is the top-level cache command.
type Cmd struct {
	Info    InfoCmd    `kong:"cmd,name='info',help='Show binary cache info'"`
	NarInfo NarInfoCmd `kong:"cmd,name='narinfo',help='Show narinfo for a store path'"`
	Closure ClosureCmd `kong:"cmd,name='closure',help='Show the closure of a store path'"`
	Tree    TreeCmd    `kong:"cmd,name='tree',help='Show the dependency tree of a store path'"`
	Diff    DiffCmd    `kong:"cmd,name='diff',help='Compare closures of two store paths'"`
	Fetch   FetchCmd   `kong:"cmd,name='fetch',help='Fetch store paths from a binary cache'"`
}

// extractHash extracts the 32-char hash from a store path or hash string.
// It handles both full store paths (/nix/store/<hash>-<name>) and bare hashes.
func extractHash(s string) string {
	h := strings.TrimPrefix(s, storepath.StoreDir+"/")
	if idx := strings.Index(h, "-"); idx > 0 {
		h = h[:idx]
	}
	return h
}

// InfoCmd prints /nix-cache-info from a binary cache.
type InfoCmd struct {
	URL string `kong:"arg,required,help='Binary cache URL'"`
}

func (cmd *InfoCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c := binarycache.New(cmd.URL)

	info, err := c.GetCacheInfo(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("StoreDir: %s\n", info.StoreDir)
	fmt.Printf("WantMassQuery: %v\n", info.WantMassQuery)
	fmt.Printf("Priority: %d\n", info.Priority)

	return nil
}

// NarInfoCmd prints the narinfo for a store path.
type NarInfoCmd struct {
	URL  string `kong:"arg,required,help='Binary cache URL'"`
	Path string `kong:"arg,required,help='Store path or hash'"`
}

func (cmd *NarInfoCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cache := binarycache.New(cmd.URL)

	ni, err := cache.GetNarInfo(ctx, extractHash(cmd.Path))
	if err != nil {
		return err
	}

	fmt.Print(ni.String())

	return nil
}

// ClosureCmd prints the closure of a store path.
type ClosureCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path to inspect'"`
}

// resolveFull resolves the full closure for the given hashes, treating all
// paths as missing (i.e. traversing everything).
func resolveFull(ctx context.Context, url string, hashes []string) ([]*narinfo.NarInfo, error) {
	c := binarycache.New(url)

	allMissing := func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}

	return c.ResolveClosure(ctx, hashes, allMissing)
}

func (cmd *ClosureCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	closure, err := resolveFull(ctx, cmd.URL, []string{extractHash(cmd.StorePath)})
	if err != nil {
		return err
	}

	var total uint64
	for _, ni := range closure {
		fmt.Printf("%s\t%d\n", ni.StorePath, ni.NarSize)
		total += ni.NarSize
	}

	fmt.Printf("total: %d paths, %d bytes\n", len(closure), total)

	return nil
}

// TreeCmd prints the dependency tree of a store path.
type TreeCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path to inspect'"`
}

func (cmd *TreeCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	closure, err := resolveFull(ctx, cmd.URL, []string{extractHash(cmd.StorePath)})
	if err != nil {
		return err
	}

	// Build reference graph: absolute store path → absolute reference paths.
	refs := make(map[string][]string, len(closure))
	inClosure := make(map[string]bool, len(closure))

	for _, ni := range closure {
		inClosure[ni.StorePath] = true
	}

	for _, ni := range closure {
		var children []string
		for _, ref := range ni.References {
			abs := storepath.StoreDir + "/" + ref
			if inClosure[abs] {
				children = append(children, abs)
			}
		}
		sort.Strings(children)
		refs[ni.StorePath] = children
	}

	// Find the root store path.
	root := ""
	for _, ni := range closure {
		if extractHash(ni.StorePath) == extractHash(cmd.StorePath) {
			root = ni.StorePath
			break
		}
	}

	if root == "" {
		return fmt.Errorf("root path not found in closure")
	}

	visited := make(map[string]bool)

	var printTree func(path, prefix string, isLast bool)
	printTree = func(path, prefix string, isLast bool) {
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		if visited[path] {
			fmt.Printf("%s%s%s [...]\n", prefix, connector, path)
			return
		}

		fmt.Printf("%s%s%s\n", prefix, connector, path)
		visited[path] = true

		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		for i, child := range refs[path] {
			printTree(child, childPrefix, i == len(refs[path])-1)
		}
	}

	// Print root separately, then recurse into its children.
	fmt.Println(root)
	visited[root] = true

	children := refs[root]
	for i, child := range children {
		printTree(child, "", i == len(children)-1)
	}

	return nil
}

// DiffCmd compares closures of two store paths.
type DiffCmd struct {
	URL   string `kong:"arg,required,help='Binary cache URL'"`
	PathA string `kong:"arg,required,help='First store path'"`
	PathB string `kong:"arg,required,help='Second store path'"`
}

func (cmd *DiffCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	closureA, err := resolveFull(ctx, cmd.URL, []string{extractHash(cmd.PathA)})
	if err != nil {
		return fmt.Errorf("resolving %s: %w", cmd.PathA, err)
	}

	closureB, err := resolveFull(ctx, cmd.URL, []string{extractHash(cmd.PathB)})
	if err != nil {
		return fmt.Errorf("resolving %s: %w", cmd.PathB, err)
	}

	setA := make(map[string]*narinfo.NarInfo, len(closureA))
	for _, ni := range closureA {
		setA[ni.StorePath] = ni
	}

	setB := make(map[string]*narinfo.NarInfo, len(closureB))
	for _, ni := range closureB {
		setB[ni.StorePath] = ni
	}

	var onlyA, onlyB, common []string

	for p := range setA {
		if _, ok := setB[p]; ok {
			common = append(common, p)
		} else {
			onlyA = append(onlyA, p)
		}
	}

	for p := range setB {
		if _, ok := setA[p]; !ok {
			onlyB = append(onlyB, p)
		}
	}

	sort.Strings(onlyA)
	sort.Strings(onlyB)
	sort.Strings(common)

	if len(onlyA) > 0 {
		fmt.Printf("Only in %s:\n", cmd.PathA)
		for _, p := range onlyA {
			fmt.Printf("  %s\t%d\n", p, setA[p].NarSize)
		}
		fmt.Println()
	}

	if len(onlyB) > 0 {
		fmt.Printf("Only in %s:\n", cmd.PathB)
		for _, p := range onlyB {
			fmt.Printf("  %s\t%d\n", p, setB[p].NarSize)
		}
		fmt.Println()
	}

	var commonSize uint64
	for _, p := range common {
		commonSize += setA[p].NarSize
	}

	fmt.Printf("In both (%d paths, %d bytes)\n", len(common), commonSize)

	return nil
}

// FetchCmd fetches store paths and their closures from a binary cache.
type FetchCmd struct {
	URL        string   `kong:"arg,required,help='Binary cache URL'"`
	StorePaths []string `kong:"arg,required,help='Store paths to fetch'"`
	Socket     string   `kong:"optional,default='/nix/var/nix/daemon-socket/socket',help='Nix daemon socket path'"`
}

func (cmd *FetchCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := daemon.Connect(cmd.Socket)
	if err != nil {
		return fmt.Errorf("connect to daemon: %w", err)
	}
	defer client.Close()

	cache := binarycache.New(cmd.URL)

	filter := func(ctx context.Context, sp string) (bool, error) {
		valid, err := client.IsValidPath(ctx, sp)
		if err != nil {
			return false, err
		}
		return !valid, nil
	}

	importer := binarycache.ImporterFunc(func(ctx context.Context, ni *narinfo.NarInfo, nar io.Reader) error {
		info := &daemon.PathInfo{
			StorePath: ni.StorePath,
			NarHash:   ni.NarHash.String(),
			NarSize:   ni.NarSize,
			CA:        ni.CA,
		}

		// Convert relative references to absolute paths.
		for _, ref := range ni.References {
			info.References = append(info.References, storepath.StoreDir+"/"+ref)
		}

		if ni.Deriver != "" {
			info.Deriver = storepath.StoreDir + "/" + ni.Deriver
		}

		for _, sig := range ni.Signatures {
			info.Sigs = append(info.Sigs, sig.String())
		}

		fmt.Printf("importing %s (%d bytes)\n", ni.StorePath, ni.NarSize)
		return client.AddToStoreNar(ctx, info, nar, false, false)
	})

	hashes := make([]string, len(cmd.StorePaths))
	for i, sp := range cmd.StorePaths {
		hashes[i] = extractHash(sp)
	}

	return cache.Substitute(ctx, hashes, filter, importer)
}
