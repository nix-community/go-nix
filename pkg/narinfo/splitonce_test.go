//nolint:testpackage
package narinfo

import (
	"fmt"
	"testing"
)

func TestSplitOnce(t *testing.T) {
	tests := []struct {
		s    string
		sep  string
		str1 string
		str2 string
		err  string
	}{
		{"hello:world", ":", "hello", "world", ""},
		{":helloworld", ":", "", "helloworld", ""},
		{"helloworld:", ":", "helloworld", "", ""},
		{"helloworld", ":", "", "", "unable to find separator ':' in helloworld"},
		{"hello:wo:rld", ":", "", "", "found separator ':' twice or more in hello:wo:rld"},
		{"hello::world", ":", "", "", "found separator ':' twice or more in hello::world"},
	}

	for _, ltest := range tests {
		// TODO: This is not necessary on Go >=1.23. Remove this assignment and use
		// test instead of ltest above.
		test := ltest

		tName := fmt.Sprintf("splitOnce(%q, %q) -> (%q, %q, %s)",
			test.s, test.sep, test.str1, test.str2, test.err)

		t.Run(tName, func(t *testing.T) {
			t.Parallel()

			str1, str2, err := splitOnce(test.s, test.sep)

			if test.err == "" && err != nil {
				t.Fatalf("expected no error but got %s", err)
			} else if test.err != "" && err == nil {
				t.Fatalf("expected an error but got none")
			} else if test.err != "" && err != nil {
				if want, got := test.err, err.Error(); want != got {
					t.Errorf("want %q got %q", want, got)
				}
			}

			if want, got := test.str1, str1; want != got {
				t.Errorf("want %q got %q", want, got)
			}

			if want, got := test.str2, str2; want != got {
				t.Errorf("want %q got %q", want, got)
			}
		})
	}
}
