package nixbase32

import (
	"encoding/base32"
	"strings"
)

const alphabet = "0123456789abcdfghijklmnpqrsvwxyz"

func EncodedLen(n int) int {
	return (n*8-1)/5 + 1
}

func Encode(buf []byte) string {
	l := EncodedLen(len(buf))
	s := ""
	for n := l - 1; n >= 0; n-- {
		b := uint(n * 5)
		i := uint(b / 8)
		j := uint(b % 8)
		c := buf[i] >> j
		if i+1 < uint(len(buf)) {
			c |= buf[i+1] << (8 - j)
		}
		s += string(alphabet[c&0x1f])
	}
	return s
}

func Decode(buf []byte, s string) (ok bool) {
	for n := 0; n < len(s); n++ {
		c := s[len(s)-n-1]
		digit := strings.IndexByte(alphabet, c)
		if digit == -1 {
			return
		}
		b := uint(n * 5)
		i := uint(b / 8)
		j := uint(b % 8)
		buf[i] |= byte(digit) << j
		if i+1 < uint(len(buf)) {
			buf[i+1] |= byte(digit) >> (8 - j)
		} else if digit>>(8-j) != 0 {
			return
		}
	}
	return true
}
