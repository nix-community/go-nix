# go-nix - Nix experiments written in go

*STATUS*: experimental

This repository holds a bunch of experiments written in Go.

## `pkg/derivation`
A parser for Nix `.drv` files

## `pkg/hash`
Methods to serialize and deserialize some of the hashes used in nix code and
`.narinfo` files.

## `pkg/nar`
A Nix ARchive (NAR) file Reader and Writer, with an interface similar to
`archive/tar` from the stdlib

## `pkg/nar/ls`
A parser for .ls files (providing an index for .nar files)

## `pkg/nar/narinfo`
A parser and generator for `.narinfo` files.

## `pkg/nixbase32`
An implementation of the slightly odd "base32" encoding that's used in Nix,
providing some of the functions in `encoding/base32.Encoding`.

## `pkg/nixpath`
A parser and regexes for Nix Store Paths.

## `pkg/wire`
Methods to parse and produce fields used in the low-level Nix wire protocol.
