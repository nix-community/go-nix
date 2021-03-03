package nixbase32

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var tt = []struct {
	dec []byte
	enc string
}{
	{[]byte{0x1f}, "0z"},
	{[]byte{
		0xd8, 0x6b, 0x33, 0x92, 0xc1, 0x20, 0x2e, 0x8f,
		0xf5, 0xa4, 0x23, 0xb3, 0x02, 0xe6, 0x28, 0x4d,
		0xb7, 0xf8, 0xf4, 0x35, 0xea, 0x9f, 0x39, 0xb5,
		0xb1, 0xb2, 0x0f, 0xd3, 0xac, 0x36, 0xdf, 0xcb},
		"1jyz6snd63xjn6skk7za6psgidsd53k05cr3lksqybi0q6936syq"},
}

func TestEncode(t *testing.T) {
	for i := range tt {
		assert.Equal(t, tt[i].enc, EncodeToString(tt[i].dec))
	}
}

func TestDecode(t *testing.T) {
	for i := range tt {
		b, err := DecodeString(tt[i].enc)

		if assert.NoError(t, err) {
			assert.Equal(t, tt[i].dec, b)
		}
	}
}

func TestDecodeInvalid(t *testing.T) {
	// this is invalid encoding, because it encodes 10 1-bytes, so the carry
	// would be 2 1-bytes
	_, err := DecodeString("zz")
	assert.Error(t, err)
}
