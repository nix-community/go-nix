package base32

import "encoding/base32"

// NixEncoding instantiates base32 with the Nix-specific alphabet
//
// omitted: E O U T
var NixEncoding = base32.NewEncoding("0123456789abcdfghijklmnpqrsvwxyz")
