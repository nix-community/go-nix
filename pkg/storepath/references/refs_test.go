package references_test

import (
	"testing"

	"github.com/nix-community/go-nix/pkg/storepath/references"
	"github.com/stretchr/testify/assert"
)

//nolint:gochecknoglobals
var cases = []struct {
	Title    string
	Chunks   []string
	Expected []string
}{
	{
		Title: "Basic",
		Chunks: []string{
			"/nix/store/knn6wc1a89c47yb70qwv56rmxylia6wx-hello-2.12/bin/hello",
		},
		Expected: []string{
			"/nix/store/knn6wc1a89c47yb70qwv56rmxylia6wx-hello-2.12",
		},
	},
	{
		Title: "PartialWrites",
		Chunks: []string{
			"/nix/store/knn6wc1a89c47yb70",
			"qwv56rmxylia6wx-hello-2.12/bin/hello",
		},
		Expected: []string{
			"/nix/store/knn6wc1a89c47yb70qwv56rmxylia6wx-hello-2.12",
		},
	},
	{
		Title: "IgnoredPaths",
		Chunks: []string{
			"/nix/store/knn6wc1a89c47yb70qwv56rmxylia6wx-hello-2.12/bin/hello",
			"/nix/store/c4pcgriqgiwz8vxrjxg7p38q3y7w3ni3-go-1.18.2/bin/go",
		},
		Expected: []string{
			"/nix/store/knn6wc1a89c47yb70qwv56rmxylia6wx-hello-2.12",
		},
	},
}

func TestReferences(t *testing.T) {
	t.Run("ScanReferences", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.Title, func(t *testing.T) {
				refScanner, err := references.NewReferenceScanner(c.Expected)
				if err != nil {
					panic(err)
				}

				for _, line := range c.Chunks {
					_, err = refScanner.Write([]byte(line))
					if err != nil {
						panic(err)
					}
				}

				assert.Equal(t, c.Expected, refScanner.References())
			})
		}
	})
}

func BenchmarkReferences(b *testing.B) {
	for _, c := range cases {
		c := c

		refScanner, err := references.NewReferenceScanner(c.Expected)
		if err != nil {
			panic(err)
		}

		chunks := make([][]byte, len(c.Chunks))
		for i, c := range c.Chunks {
			chunks[i] = []byte(c)
		}

		b.Run(c.Title, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, chunk := range chunks {
					_, err = refScanner.Write(chunk)
					if err != nil {
						panic(err)
					}
				}
			}

			assert.Equal(b, c.Expected, refScanner.References())
		})
	}
}
