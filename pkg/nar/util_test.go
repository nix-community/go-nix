package nar_test

import (
	"fmt"
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/stretchr/testify/assert"
)

func TestLexicographicallyOrdered(t *testing.T) {
	testCases := []struct {
		path1    string
		path2    string
		expected bool
	}{
		{
			path1:    "/foo",
			path2:    "/foo",
			expected: true,
		},
		{
			path1:    "/fooa",
			path2:    "/foob",
			expected: true,
		},
		{
			path1:    "/foob",
			path2:    "/fooa",
			expected: false,
		},
		{
			path1:    "/cmd/structlayout/main.go",
			path2:    "/cmd/structlayout-optimize",
			expected: true,
		},
		{
			path1:    "/cmd/structlayout-optimize",
			path2:    "/cmd/structlayout-ao/main.go",
			expected: false,
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			result := nar.PathIsLexicographicallyOrdered(testCase.path1, testCase.path2)
			assert.Equal(t, result, testCase.expected)
		})
	}
}
