package nar

import "strings"

// IsValidNodeName checks the name of a node
// it may not contain null bytes or slashes.
func IsValidNodeName(nodeName string) bool {
	return !strings.Contains(nodeName, "/") && !strings.ContainsAny(nodeName, "\u0000")
}

// PathIsLexicographicallyOrdered checks if two paths are lexicographically ordered component by component.
func PathIsLexicographicallyOrdered(path1 string, path2 string) bool {
	if path1 <= path2 {
		return true
	}

	// n is the lower number of characters of the two paths.
	var n int
	if len(path1) < len(path2) {
		n = len(path1)
	} else {
		n = len(path2)
	}

	for i := 0; i < n; i++ {
		if path1[i] == path2[i] {
			continue
		}

		if path1[i] == '/' && path2[i] != '/' {
			return true
		}

		return path1[i] < path2[i]
	}

	// Cover cases like where path1 is a prefix of path2 (path1=/arp-foo path2=/arp)
	return len(path2) >= len(path1)
}
