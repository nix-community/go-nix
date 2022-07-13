package store_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/nix-community/go-nix/pkg/nar/narinfo"
	"github.com/nix-community/go-nix/pkg/store"
	"github.com/nix-community/go-nix/pkg/store/chunkstore"
	"github.com/stretchr/testify/assert"
)

//nolint: gochecknoglobals
var strNarinfoSampleWithoutFileFields = `
StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz
Compression: xz
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Deriver: 10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv
Sig: cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==
Sig: hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA==
` // TODO: dedup

func getContentFromInsideNar(narPath string, path string) []byte {
	f, err := os.Open(narPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	nr, err := nar.NewReader(f)
	if err != nil {
		panic(err)
	}

	for {
		header, err := nr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				panic(fmt.Sprintf("couldn't find %v in nar", path))
			}

			panic(err)
		}

		if header.Path == path {
			var buf bytes.Buffer

			_, err := io.Copy(&buf, nr)
			if err != nil {
				panic(err)
			}

			return buf.Bytes()
		}
	}
}

func TestFromNarInfo(t *testing.T) {
	narPath := "../../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar"

	f, err := os.Open(narPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSampleWithoutFileFields))
	assert.NoError(t, err)

	cs, err := chunkstore.NewBadgerMemoryStore("sha2-256")
	if err != nil {
		panic(err)
	}

	pi, err := store.FromNarinfo(context.Background(), ni, f, cs)
	assert.NoError(t, err)

	t.Run("check PathInfo", func(t *testing.T) {
		assert.Equal(t, ni.StorePath, pi.OutputName)
		assert.Equal(t, ni.References, pi.References)

		assert.Equal(t, []*store.DirectoryEntry{
			{Path: "/"},
			{Path: "/bin"},
			{Path: "/share"},
			{Path: "/share/man"},
			{Path: "/share/man/man1"},
			{Path: "/share/man/man5"},
			{Path: "/share/man/man8"},
		}, pi.Directories)

		assert.Equal(t, []*store.SymlinkEntry{
			{Path: "/bin/dnsdomainname", Target: "hostname"},
			{Path: "/bin/domainname", Target: "hostname"},
			{Path: "/bin/nisdomainname", Target: "hostname"},
			{Path: "/bin/ypdomainname", Target: "hostname"},
			{Path: "/sbin", Target: "bin"},
		}, pi.Symlinks)

		// This is the expected []*store.RegularEntry, omitting the Chunks,
		// because it's too much pain to write.
		ttRegulars := []*store.RegularEntry{
			{Path: "/bin/arp", Executable: true},
			{Path: "/bin/hostname", Executable: true},
			{Path: "/bin/ifconfig", Executable: true},
			{Path: "/bin/nameif", Executable: true},
			{Path: "/bin/netstat", Executable: true},
			{Path: "/bin/plipconfig", Executable: true},
			{Path: "/bin/rarp", Executable: true},
			{Path: "/bin/route", Executable: true},
			{Path: "/bin/slattach", Executable: true},
			{Path: "/share/man/man1/dnsdomainname.1.gz", Executable: false},
			{Path: "/share/man/man1/domainname.1.gz", Executable: false},
			{Path: "/share/man/man1/hostname.1.gz", Executable: false},
			{Path: "/share/man/man1/nisdomainname.1.gz", Executable: false},
			{Path: "/share/man/man1/ypdomainname.1.gz", Executable: false},
			{Path: "/share/man/man5/ethers.5.gz", Executable: false},
			{Path: "/share/man/man8/arp.8.gz", Executable: false},
			{Path: "/share/man/man8/ifconfig.8.gz", Executable: false},
			{Path: "/share/man/man8/nameif.8.gz", Executable: false},
			{Path: "/share/man/man8/netstat.8.gz", Executable: false},
			{Path: "/share/man/man8/plipconfig.8.gz", Executable: false},
			{Path: "/share/man/man8/rarp.8.gz", Executable: false},
			{Path: "/share/man/man8/route.8.gz", Executable: false},
			{Path: "/share/man/man8/slattach.8.gz", Executable: false},
		}

		// Check Path and Executable fields for equality.
		// Assemble the chunk data and check for equality too.
		for i, tRegular := range ttRegulars {
			assert.Equal(t, tRegular.Path, pi.Regulars[i].Path)
			assert.Equal(t, tRegular.Executable, pi.Regulars[i].Executable)

			t.Run("assemble "+tRegular.Path, func(t *testing.T) {
				var assembledContents []byte
				for _, chunkMeta := range pi.Regulars[i].Chunks {
					// query the chunk store for the data
					chunkData, err := cs.Get(context.Background(), chunkMeta.Identifier)
					if err != nil {
						panic(err)
					}
					// check the size field
					assert.Equal(t, chunkMeta.Size, uint64(len(chunkData)), "size field needs to equal actual chunk size")
					assembledContents = append(assembledContents, chunkData...)
				}

				expectedContents := getContentFromInsideNar(narPath, tRegular.Path)
				assert.Equal(t, expectedContents, assembledContents, "chunks assembled together need to be equal to nar content")
			})
		}
	})
}
