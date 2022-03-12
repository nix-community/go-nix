// Package tar implements access to .nar files.
//
// Nix Archive (nar) is a file format for storing a directory or a single file
// in a binary reproducible format. This is the format that is being used to
// pack and distribute Nix build results. It doesn't store any timestamps or
// similar fields available in conventional filesystems. .nar files can be read
// and written in a streaming manner.
package nar
