package nar

import "strings"

// IsValidNodeName checks the name of a node
// it may not contain null bytes or slashes.
func IsValidNodeName(nodeName string) bool {
	return !strings.Contains(nodeName, "/") && !strings.ContainsAny(nodeName, "\u0000")
}

// PathIsLexicographicallyOrdered checks if two paths are lexicographically ordered component by component.
func PathIsLexicographicallyOrdered(path1 string, path2 string) bool {
	if path1 > path2 {
		path1Segments := strings.Split(path1, "/")
		path2Segments := strings.Split(path2, "/")

		// n is the lower number of segments of the two paths.
		var n int
		if len(path1Segments) < len(path2Segments) {
			n = len(path1Segments)
		} else {
			n = len(path2Segments)
		}

		// check all segments individually
		for i := 0; i < n; i++ {
			if !(path1Segments[i] <= path2Segments[i]) {
				return false
			}
		}
	}

	return true
}
