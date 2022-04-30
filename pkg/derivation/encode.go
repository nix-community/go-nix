package derivation

import (
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
func escapeString(s string) string {
	s = stringEscaper.Replace(s)

	return quoteString(s)
}

// Adds quotation marks around a string.
// This is primarily meant for non-user provided strings.
func quoteString(s string) string {
	buf := make([]byte, len(s)+2)

	buf[0] = '"'

	for i := 0; i < len(s); i++ {
		buf[i+1] = s[i]
	}

	buf[len(s)+1] = '"'

	return string(buf)
}

// Encode a list of elements staring with `opening` character and ending with a `closing` character.
func encodeArray(opening byte, closing byte, quote bool, elems ...string) string {
	if len(elems) == 0 {
		return string([]byte{opening, closing})
	}

	n := 3 * (len(elems) - 1)
	if quote {
		n += 2
	}

	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var b strings.Builder

	b.Grow(n)
	b.WriteByte(opening)

	writeElem := func(s string) {
		if quote {
			b.WriteByte('"')
		}

		b.WriteString(s)

		if quote {
			b.WriteByte('"')
		}
	}

	writeElem(elems[0])

	for _, s := range elems[1:] {
		b.WriteByte(',')
		writeElem(s)
	}

	b.WriteByte(closing)

	return b.String()
}
