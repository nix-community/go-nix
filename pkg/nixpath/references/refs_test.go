package references_test

import (
	"testing"

	"github.com/nix-community/go-nix/pkg/nixpath/references"
	"github.com/stretchr/testify/assert"
)

func TestReferences(t *testing.T) {
	cases := []struct {
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
