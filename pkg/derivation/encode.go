package derivation

import (
	"bytes"
	"strings"
)

// nolint:gochecknoglobals
var stringEscaper = strings.NewReplacer(
	"\\", "\\\\",
	"\n", "\\n",
	"\r", "\\r",
	"\t", "\\t",
	"\"", "\\\"",
)

// Escapes user provided values such as derivation attributes.
// These may contain special characters such as newlines, tabs, backslashes and so on.
func escapeString(s string) []byte {
	s = stringEscaper.Replace(s)

	return quoteString(s)
}

// Adds quotation marks around a string.
// This is primarily meant for non-user provided strings.
func quoteString(s string) []byte {
	return []byte("\"" + s + "\"")
}

// Encode a list of elements staring with `opening` character and ending with a `closing` character.
func encodeArray(opening byte, closing byte, quote bool, elems ...[]byte) []byte {
	if len(elems) == 0 {
		return []byte{opening, closing}
	}

	n := 2 + (len(elems) - 1) // one byte per item where i > 1
	if quote {
		n += 2 * len(elems) // 2 extra bytes per quoted item
	}

	for i := 0; i < len(elems); i++ {
		n += len(elems[i]) // Element length
	}

	var buf bytes.Buffer

	buf.Grow(n)
	buf.WriteByte(opening)

	for i, b := range elems {
		if i > 0 {
			buf.WriteByte(',')
		}

		if quote {
			buf.WriteByte('"')
		}

		buf.Write(b)

		if quote {
			buf.WriteByte('"')
		}
	}

	buf.WriteByte(closing)

	return buf.Bytes()
}

func encodeArrayStrings(opening byte, closing byte, quote bool, elems ...string) []byte {
	if len(elems) == 0 {
		return []byte{opening, closing}
	}

	n := 2 + (len(elems) - 1) // one byte per item where i > 1
	if quote {
		n += 2 * len(elems) // 2 extra bytes per quoted item
	}

	for i := 0; i < len(elems); i++ {
		n += len(elems[i]) // Element length
	}

	var buf bytes.Buffer

	buf.Grow(n)
	buf.WriteByte(opening)

	for i, b := range elems {
		if i > 0 {
			buf.WriteByte(',')
		}

		if quote {
			buf.WriteByte('"')
		}

		buf.Write([]byte(b))

		if quote {
			buf.WriteByte('"')
		}
	}

	buf.WriteByte(closing)

	return buf.Bytes()
}
