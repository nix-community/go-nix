package nixbase32_test

import (
	"bytes"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	//nolint:revive
	. "github.com/nix-community/go-nix/pkg/nixbase32"
	"github.com/stretchr/testify/assert"
)

//nolint:gochecknoglobals
var tests = []struct {
	dec []byte
	enc string
}{
	{[]byte{}, ""},
	{[]byte{0x1f}, "0z"},
	{
		[]byte{
			0xd8, 0x6b, 0x33, 0x92, 0xc1, 0x20, 0x2e, 0x8f,
			0xf5, 0xa4, 0x23, 0xb3, 0x02, 0xe6, 0x28, 0x4d,
			0xb7, 0xf8, 0xf4, 0x35, 0xea, 0x9f, 0x39, 0xb5,
			0xb1, 0xb2, 0x0f, 0xd3, 0xac, 0x36, 0xdf, 0xcb,
		},
		"1jyz6snd63xjn6skk7za6psgidsd53k05cr3lksqybi0q6936syq",
	},
}

//nolint:gochecknoglobals
var invalidEncodings = []string{
	// invalid character
	"0t",
	// this is invalid encoding, because it encodes 10 1-bytes, so the carry
	// would be 2 1-bytes
	"zz",
	// this is an even more specific example - it'd decode as 00000000 11
	"c0",
}

func TestEncode(t *testing.T) {
	for _, test := range tests {
		got := make([]byte, EncodedLen(len(test.dec)))
		Encode(got, test.dec)

		if string(got) != test.enc {
			t.Errorf("after Encode(dst, %q), dst = %q; want %q", test.dec, got, test.enc)
		}
	}
}

func TestEncodeToString(t *testing.T) {
	for _, test := range tests {
		if got := EncodeToString(test.dec); got != test.enc {
			t.Errorf("EncodeToString(%q) = %q; want %q", test.dec, got, test.enc)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, test := range tests {
		got := make([]byte, DecodedLen(len(test.enc)))
		n, err := Decode(got, []byte(test.enc))
		got = got[:n]

		if err != nil || !bytes.Equal(got, test.dec) {
			t.Errorf(
				"Decode(dst, %q) = %d, %v (dst=%02x); want %d, <nil> (dst=%02x)",
				test.enc, n, err, got, len(test.dec), test.dec,
			)
		}
	}

	for _, bad := range invalidEncodings {
		n, err := Decode(make([]byte, DecodedLen(len(bad))), []byte(bad))
		if err == nil {
			t.Errorf("Decode(dst, %q) = %d, <nil>; want _, <error>", bad, n)
		}
	}
}

func TestDecodeString(t *testing.T) {
	for _, test := range tests {
		got, err := DecodeString(test.enc)
		if err != nil || !bytes.Equal(got, test.dec) {
			t.Errorf("DecodeString(%q) = %02x, %v; want %02x, <nil>", test.enc, got, err, test.dec)
		}
	}

	for _, bad := range invalidEncodings {
		if got, err := DecodeString(bad); err == nil {
			t.Errorf("DecodeString(%q) = %q, <nil>; want _, <error>", bad, got)
		}
	}
}

func TestEncodedLen(t *testing.T) {
	for _, test := range tests {
		n := len(test.dec)
		if got, want := EncodedLen(n), len(test.enc); got != want {
			t.Errorf("EncodedLen(%d) = %d; want %d", n, got, want)
		}
	}
}

func TestDecodedLen(t *testing.T) {
	for _, test := range tests {
		n := len(test.enc)
		if got, want := DecodedLen(n), len(test.dec); got != want {
			t.Errorf("DecodedLen(%d) = %d; want %d", n, got, want)
		}
	}
}

func TestValidateString(t *testing.T) {
	for _, test := range tests {
		if err := ValidateString(test.enc); err != nil {
			t.Errorf("ValidateString(%q) = %v; want <nil>", test.enc, err)
		}
	}

	for _, enc := range invalidEncodings {
		if err := ValidateString(enc); err == nil {
			t.Errorf("ValidateString(%q) = nil; want <error>", enc)
		}
	}
}

func TestIs(t *testing.T) {
	for c := int16(0); c <= 0xff; c++ {
		got := Is(byte(c))
		want := strings.IndexByte(Alphabet, byte(c)) != -1

		if got != want {
			t.Errorf("Is(%q) = %t; want %t", byte(c), got, want)
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	sizes := []int{32, 64, 128}

	for _, s := range sizes {
		bytes := make([]byte, s)

		rand.Read(bytes) //nolint:gosec,staticcheck

		buf := make([]byte, EncodedLen(s))

		b.Run(strconv.Itoa(s), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(bytes)))

			for i := 0; i < b.N; i++ {
				Encode(buf, bytes)
			}
		})
	}
}

func BenchmarkEncodeToString(b *testing.B) {
	sizes := []int{32, 64, 128}

	for _, s := range sizes {
		bytes := make([]byte, s)
		rand.Read(bytes) //nolint:gosec,staticcheck

		b.Run(strconv.Itoa(s), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(bytes)))

			for i := 0; i < b.N; i++ {
				EncodeToString(bytes)
			}
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	sizes := []int{32, 64, 128}

	for _, s := range sizes {
		bytes := make([]byte, s)

		rand.Read(bytes) //nolint:gosec,staticcheck

		input := make([]byte, EncodedLen(s))
		Encode(input, bytes)

		b.Run(strconv.Itoa(s), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for i := 0; i < b.N; i++ {
				if _, err := Decode(bytes, input); err != nil {
					b.Fatal("error: %w", err)
				}
			}
		})
	}
}

func BenchmarkDecodeString(b *testing.B) {
	sizes := []int{32, 64, 128}

	for _, s := range sizes {
		bytes := make([]byte, s)
		rand.Read(bytes) //nolint:gosec,staticcheck
		input := EncodeToString(bytes)

		b.Run(strconv.Itoa(s), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for i := 0; i < b.N; i++ {
				_, err := DecodeString(input)
				if err != nil {
					b.Fatal("error: %w", err)
				}
			}
		})
	}
}

func BenchmarkValidateString(b *testing.B) {
	sizes := []int{32, 64, 128}

	for _, s := range sizes {
		bytes := make([]byte, s)
		rand.Read(bytes) //nolint:gosec,staticcheck
		input := EncodeToString(bytes)

		b.Run(strconv.Itoa(s), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for i := 0; i < b.N; i++ {
				err := ValidateString(input)
				if err != nil {
					b.Fatal("error: %w", err)
				}
			}
		})
	}
}

func FuzzDecodeString(f *testing.F) {
	for _, test := range tests {
		f.Add(test.enc)
	}

	f.Fuzz(func(t *testing.T, enc1 string) {
		dec, err := DecodeString(enc1)
		if err != nil {
			t.Skip()
		}

		enc2 := EncodeToString(dec)

		assert.Equal(t, enc1, enc2)
	})
}

func FuzzEncodeToString(f *testing.F) {
	for _, test := range tests {
		f.Add(test.dec)
	}

	f.Fuzz(func(t *testing.T, dec1 []byte) {
		enc1 := EncodeToString(dec1)

		dec2, err := DecodeString(enc1)
		if err != nil {
			t.Skip()
		}

		enc2 := EncodeToString(dec2)

		assert.Equal(t, enc1, enc2)
	})
}
