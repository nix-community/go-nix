package cache

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/nix-community/go-nix/pkg/binarycache"
	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/storepath"
)

// Cmd is the top-level cache command.
type Cmd struct {
	Info  InfoCmd  `kong:"cmd,name='info',help='Show binary cache info'"`
	Fetch FetchCmd `kong:"cmd,name='fetch',help='Fetch a store path from a binary cache'"`
}

// InfoCmd prints /nix-cache-info from a binary cache.
type InfoCmd struct {
	URL string `kong:"arg,required,help='Binary cache URL'"`
}

func (cmd *InfoCmd) Run() error {
	c := binarycache.New(cmd.URL)

	info, err := c.GetCacheInfo(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("StoreDir: %s\n", info.StoreDir)
	fmt.Printf("WantMassQuery: %v\n", info.WantMassQuery)
	fmt.Printf("Priority: %d\n", info.Priority)

	return nil
}

// FetchCmd fetches a store path and its closure from a binary cache.
type FetchCmd struct {
	URL       string `kong:"arg,required,help='Binary cache URL'"`
	StorePath string `kong:"arg,required,help='Store path to fetch'"`
	Socket    string `kong:"optional,default='/nix/var/nix/daemon-socket/socket',help='Nix daemon socket path'"`
}

func (cmd *FetchCmd) Run() error {
	ctx := context.Background()

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

	// Extract hash from store path: /nix/store/<hash>-<name> -> <hash>
	hash := strings.TrimPrefix(cmd.StorePath, storepath.StoreDir+"/")
	if idx := strings.Index(hash, "-"); idx > 0 {
		hash = hash[:idx]
	}

	return cache.Substitute(ctx, []string{hash}, filter, importer)
}
