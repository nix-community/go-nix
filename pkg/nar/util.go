package nar

import "strings"

// isValidNodeName checks the name of a node
// it may not contain null bytes or slashes.
func isValidNodeName(nodeName string) bool {
	return !strings.Contains(nodeName, "/") && !strings.ContainsAny(nodeName, "\000/")
}
