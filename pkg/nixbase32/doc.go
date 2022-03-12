// Package nixbase32 implements the slightly odd "base32" encoding that's used
// in Nix.

// Nix uses a custom alphabet. Contrary to other implementations (RFC4648),
// encoding to "nix base32" also reads in characters in reverse order (and
// doesn't use any padding), which makes adopting encoding/base32 hard.
// This package provides some of the functions defined in
// encoding/base32.Encoding.

package nixbase32
