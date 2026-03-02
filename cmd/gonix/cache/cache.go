package cache

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/nix-community/go-nix/pkg/binarycache"
	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/narinfo/signature"
	"github.com/nix-community/go-nix/pkg/storepath"
)

// Cmd is the top-level cache command.
type Cmd struct {
	Info       InfoCmd       `kong:"cmd,name='info',help='Show binary cache info'"`
	NarInfo    NarInfoCmd    `kong:"cmd,name='narinfo',help='Show narinfo for a store path'"`
	Closure    ClosureCmd    `kong:"cmd,name='closure',help='Show the closure of a store path'"`
	Tree       TreeCmd       `kong:"cmd,name='tree',help='Show the dependency tree of a store path'"`
	Diff       DiffCmd       `kong:"cmd,name='diff',help='Compare closures of two store paths'"`
	WhyDepends WhyDependsCmd `kong:"cmd,name='why-depends',help='Show why a store path depends on another'"`
	Verify     VerifyCmd     `kong:"cmd,name='verify',help='Verify narinfo signatures in a closure'"`
	Size       SizeCmd       `kong:"cmd,name='size',help='Show closure size breakdown'"`
	Log        LogCmd        `kong:"cmd,name='log',help='Fetch build log for a store path'"`
	Dot        DotCmd        `kong:"cmd,name='dot',help='Emit dependency graph in Graphviz DOT format'"`
	Check      CheckCmd      `kong:"cmd,name='check',help='Check if store paths exist in a binary cache'"`
	Fetch      FetchCmd      `kong:"cmd,name='fetch',help='Fetch store paths from a binary cache'"`
	NarLs      NarLsCmd      `kong:"cmd,name='nar-ls',help='List files inside a NAR from a binary cache'"`
	NarCat     NarCatCmd     `kong:"cmd,name='nar-cat',help='Print file contents from a NAR in a binary cache'"`
	Serve      ServeCmd      `kong:"cmd,name='serve',help='Serve the local Nix store as an HTTP binary cache'"`
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

// WhyDependsCmd finds the shortest dependency chain between two store paths.
type WhyDependsCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path to inspect'"`
	Dep       string `kong:"arg,required,help='Dependency to find'"`
}

func (cmd *WhyDependsCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	closure, err := resolveFull(ctx, cmd.URL, []string{extractHash(cmd.StorePath)})
	if err != nil {
		return err
	}

	// Build reference graph and locate root + target.
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

	root := ""
	target := ""
	depHash := extractHash(cmd.Dep)

	for _, ni := range closure {
		h := extractHash(ni.StorePath)
		if h == extractHash(cmd.StorePath) {
			root = ni.StorePath
		}
		if h == depHash {
			target = ni.StorePath
		}
	}

	if root == "" {
		return fmt.Errorf("root path not found in closure")
	}

	if target == "" {
		return fmt.Errorf("%s is not in the closure of %s", cmd.Dep, cmd.StorePath)
	}

	// BFS to find shortest path from root to target.
	parent := map[string]string{root: ""}
	queue := []string{root}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur == target {
			break
		}

		for _, child := range refs[cur] {
			if _, seen := parent[child]; !seen {
				parent[child] = cur
				queue = append(queue, child)
			}
		}
	}

	if _, reached := parent[target]; !reached {
		return fmt.Errorf("%s is not reachable from %s", cmd.Dep, cmd.StorePath)
	}

	// Reconstruct path from root to target.
	var chain []string
	for cur := target; cur != ""; cur = parent[cur] {
		chain = append(chain, cur)
	}

	// Reverse to get root → target order.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	// Print the chain with box-drawing indentation.
	for i, path := range chain {
		if i == 0 {
			fmt.Println(path)
		} else {
			indent := strings.Repeat("    ", i-1)
			fmt.Printf("%s└── %s\n", indent, path)
		}
	}

	return nil
}

// VerifyCmd verifies narinfo signatures in a closure.
type VerifyCmd struct {
	URL       string   `kong:"arg,required,help='Binary cache URL'"`
	StorePath string   `kong:"arg,required,help='Store path to verify'"`
	Keys      []string `kong:"required,name='key',short='k',help='Public keys (name:base64)'"`
}

func (cmd *VerifyCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pubKeys := make([]signature.PublicKey, len(cmd.Keys))
	for i, k := range cmd.Keys {
		pk, err := signature.ParsePublicKey(k)
		if err != nil {
			return fmt.Errorf("parsing public key %q: %w", k, err)
		}
		pubKeys[i] = pk
	}

	closure, err := resolveFull(ctx, cmd.URL, []string{extractHash(cmd.StorePath)})
	if err != nil {
		return err
	}

	passed := 0
	failed := 0

	for _, ni := range closure {
		fp := ni.Fingerprint()
		matched := ""

		// Find which key name matched by checking individually.
		for _, key := range pubKeys {
			for _, sig := range ni.Signatures {
				if key.Verify(fp, sig) {
					matched = key.Name
					break
				}
			}
			if matched != "" {
				break
			}
		}

		if matched != "" {
			fmt.Printf("PASS  %s  (%s)\n", ni.StorePath, matched)
			passed++
		} else {
			fmt.Printf("FAIL  %s  (no matching signature)\n", ni.StorePath)
			failed++
		}
	}

	fmt.Printf("\n%d/%d passed\n", passed, passed+failed)

	if failed > 0 {
		return fmt.Errorf("%d paths failed signature verification", failed)
	}

	return nil
}

// SizeCmd shows closure size breakdown.
type SizeCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path to inspect'"`
}

func (cmd *SizeCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	closure, err := resolveFull(ctx, cmd.URL, []string{extractHash(cmd.StorePath)})
	if err != nil {
		return err
	}

	var totalNar, totalFile uint64

	for _, ni := range closure {
		fmt.Printf("%s    nar:%d  download:%d  %s\n",
			ni.StorePath, ni.NarSize, ni.FileSize, ni.Compression)
		totalNar += ni.NarSize
		totalFile += ni.FileSize
	}

	ratio := uint64(0)
	if totalNar > 0 {
		ratio = totalFile * 100 / totalNar
	}

	fmt.Printf("\ntotal: %d paths, nar:%d, download:%d (%d%% ratio)\n",
		len(closure), totalNar, totalFile, ratio)

	return nil
}

// LogCmd fetches the build log for a store path from a binary cache.
type LogCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path to fetch build log for'"`
}

func (cmd *LogCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	basename := strings.TrimPrefix(cmd.StorePath, storepath.StoreDir+"/")
	u := cmd.URL + "/log/" + basename

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("no build log available for %s", cmd.StorePath)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d fetching build log", resp.StatusCode)
	}

	_, err = io.Copy(os.Stdout, resp.Body)

	return err
}

// DotCmd emits the dependency graph in Graphviz DOT format.
type DotCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path to inspect'"`
}

func (cmd *DotCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	closure, err := resolveFull(ctx, cmd.URL, []string{extractHash(cmd.StorePath)})
	if err != nil {
		return err
	}

	// Build reference graph.
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

	fmt.Println("digraph closure {")
	fmt.Println("    rankdir=LR;")

	for _, ni := range closure {
		from := strings.TrimPrefix(ni.StorePath, storepath.StoreDir+"/")
		for _, child := range refs[ni.StorePath] {
			to := strings.TrimPrefix(child, storepath.StoreDir+"/")
			fmt.Printf("    %q -> %q;\n", from, to)
		}
	}

	fmt.Println("}")

	return nil
}

// CheckCmd tests whether store paths exist in a binary cache.
type CheckCmd struct {
	URL        string   `kong:"arg,required,help='Binary cache URL'"`
	StorePaths []string `kong:"arg,required,help='Store paths to check'"`
}

func (cmd *CheckCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cache := binarycache.New(cmd.URL)

	hits := 0

	for _, sp := range cmd.StorePaths {
		_, err := cache.GetNarInfo(ctx, extractHash(sp))
		if err != nil {
			fmt.Printf("MISS  %s\n", sp)
		} else {
			fmt.Printf("HIT   %s\n", sp)
			hits++
		}
	}

	fmt.Printf("\n%d/%d available\n", hits, len(cmd.StorePaths))

	if hits < len(cmd.StorePaths) {
		return fmt.Errorf("%d paths not found in cache", len(cmd.StorePaths)-hits)
	}

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

// headerLineString returns a one-line string describing a NAR header.
func headerLineString(hdr *nar.Header) string {
	var sb strings.Builder

	sb.WriteString(hdr.FileInfo().Mode().String())
	sb.WriteString(" ")
	sb.WriteString(hdr.Path)

	if hdr.Size > 0 {
		sb.WriteString(fmt.Sprintf(" (%v bytes)", hdr.Size))
	}

	if hdr.LinkTarget != "" {
		sb.WriteString(" -> ")
		sb.WriteString(hdr.LinkTarget)
	}

	sb.WriteString("\n")

	return sb.String()
}

// NarLsCmd lists files inside a NAR from a binary cache.
type NarLsCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path or hash'"`
	Path      string `kong:"arg,optional,default='/',help='Path inside the NAR'"`
	Recursive bool   `kong:"short='R',help='List recursively'"`
}

func (cmd *NarLsCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cache := binarycache.New(cmd.URL)

	ni, err := cache.GetNarInfo(ctx, extractHash(cmd.StorePath))
	if err != nil {
		return err
	}

	rc, err := cache.GetNar(ctx, ni)
	if err != nil {
		return err
	}
	defer rc.Close()

	nr, err := nar.NewReader(rc)
	if err != nil {
		return err
	}

	for {
		hdr, err := nr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if strings.HasPrefix(hdr.Path, cmd.Path) {
			remainder := hdr.Path[len(cmd.Path):]
			if cmd.Recursive || !strings.Contains(remainder, "/") {
				print(headerLineString(hdr))
			}
		} else {
			if hdr.Path > cmd.Path {
				return nil
			}
		}
	}
}

// NarCatCmd prints file contents from a NAR in a binary cache.
type NarCatCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path or hash'"`
	Path      string `kong:"arg,required,help='Path inside the NAR'"`
}

func (cmd *NarCatCmd) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cache := binarycache.New(cmd.URL)

	ni, err := cache.GetNarInfo(ctx, extractHash(cmd.StorePath))
	if err != nil {
		return err
	}

	rc, err := cache.GetNar(ctx, ni)
	if err != nil {
		return err
	}
	defer rc.Close()

	nr, err := nar.NewReader(rc)
	if err != nil {
		return err
	}

	for {
		hdr, err := nr.Next()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("requested path not found")
			}
			return err
		}

		if hdr.Path == cmd.Path {
			if hdr.Type != nar.TypeRegular {
				return fmt.Errorf("unable to cat non-regular file")
			}

			w := bufio.NewWriter(os.Stdout)

			_, err := io.Copy(w, nr)
			if err != nil {
				return err
			}

			return w.Flush()
		}
	}
}
