package nixbase32

import (
	"fmt"
	"strings"
)

const alphabet = "0123456789abcdfghijklmnpqrsvwxyz"

// EncodedLen returns the length in bytes of the base32 encoding of an input
// buffer of length n.
func EncodedLen(n int) int {
	return (n*8-1)/5 + 1
}

// DecodedLen returns the length in bytes of the decoded data
// corresponding to n bytes of base32-encoded data.
// If we have bits that don't fit into here, they are padding and must
// be 0.
func DecodedLen(n int) int {
	return (n * 5) / 8
}

// EncodeToString returns the nixbase32 encoding of src.
func EncodeToString(src []byte) string {
	l := EncodedLen(len(src))

	var dst strings.Builder
	dst.Grow(l)

	for n := l - 1; n >= 0; n-- {
		b := uint(n * 5)
		i := uint(b / 8)
		j := uint(b % 8)

		c := src[i] >> j

		if i+1 < uint(len(src)) {
			c |= src[i+1] << (8 - j)
		}

		dst.WriteByte(alphabet[c&0x1f])
	}
	return dst.String()
}

// DecodeString returns the bytes represented by the nixbase32 string s.
func DecodeString(s string) ([]byte, error) {
	dst := make([]byte, DecodedLen(len(s)))
	for n := 0; n < len(s); n++ {
		c := s[len(s)-n-1]
		digit := strings.IndexByte(alphabet, c)
		if digit == -1 {
			return nil, fmt.Errorf("character %v not in alphabet!", c)
		}

		b := uint(n * 5)
		i := uint(b / 8)
		j := uint(b % 8)

		// OR the main pattern
		dst[i] |= byte(digit) << j

		// calculate the "carry pattern"
		carry := byte(digit) >> (8 - j)

		// if we're at the end of dst…
		if i == uint(len(dst)-1) {
			// but have a nonzero carry, the encoding is invalid.
			if carry != 0 {
				return nil, fmt.Errorf("invalid encoding")
			}
		} else {
			dst[i+1] |= carry
		}
	}
	return dst, nil
}
