package derivation

import (
	"bytes"
)

// Escapes user provided values such as derivation attributes.
// These may contain special characters such as newlines, tabs, backslashes and so on.
func escapeString(s string) []byte {
	return quoteString(s)
}

// Adds quotation marks around a string.
// This is primarily meant for non-user provided strings.
func quoteString(s string) []byte {
	buf := make([]byte, len(s)+2)

	buf[0] = '"'

	for i := 0; i < len(s); i++ {
		buf[i+1] = s[i]
	}

	buf[len(s)+1] = '"'

	return buf
}

// Convert a slice of strings to a slice of byte slices.
func stringsToBytes(elems []string) [][]byte {
	b := make([][]byte, len(elems))

	for i, s := range elems {
		b[i] = []byte(s)
	}

	return b
}

// Encode a list of elements staring with `opening` character and ending with a `closing` character.
func encodeArray(opening byte, closing byte, quote bool, elems ...[]byte) []byte {
	if len(elems) == 0 {
		return []byte{opening, closing}
	}

	n := 3 * (len(elems) - 1)
	if quote {
		n += 2
	}

	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var buf bytes.Buffer

	buf.Grow(n)
	buf.WriteByte(opening)

	writeElem := func(b []byte) {
		if quote {
			buf.WriteByte('"')
		}

		buf.Write(b)

		if quote {
			buf.WriteByte('"')
		}
	}

	writeElem(elems[0])

	for _, s := range elems[1:] {
		buf.WriteByte(',')
		writeElem(s)
	}

	buf.WriteByte(closing)

	return buf.Bytes()
}
