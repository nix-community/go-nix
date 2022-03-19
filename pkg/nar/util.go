package nar

import "strings"

// IsValidNodeName checks the name of a node
// it may not contain null bytes or slashes.
func IsValidNodeName(nodeName string) bool {
	return !strings.Contains(nodeName, "/") && !strings.ContainsAny(nodeName, "\u0000")
}
