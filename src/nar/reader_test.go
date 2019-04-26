package nar_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimbatm/go-nix/src/nar"
)

func TestReader(t *testing.T) {
	f, err := os.Open("fixtures/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar")
	if !assert.NoError(t, err) {
		return
	}

	p := nar.NewReader(f)

	// Get top-level directory
	hdr, err := p.Next()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, &nar.Header{
		Type: nar.TypeDirectory,
	}, hdr)

	// Get first entry
	hdr, err = p.Next()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, &nar.Header{
		Type: nar.TypeDirectory,
		Name: "bin",
	}, hdr)

	// Get second entry
	hdr, err = p.Next()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, &nar.Header{
		Type: nar.TypeRegular,
		Name: "bin/arp",
		Executable: true,
		Size: 55288,
	}, hdr)

	hdr, err = p.Next()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, &nar.Header{
		Type: nar.TypeRegular,
		Name: "bin/arp",
		Executable: true,
		Size: 55288,
	}, hdr)
}
