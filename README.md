# go-nix - Nix experiments written in go

_STATUS_: experimental

This repository holds a bunch of experiments written in Go.

## `cmd/gonix`

A command line entrypoint called `gonix`, currently implementing the nar
{cat,dump-path,ls} commands.

They're not meant to be 100% compatible, but are documented in the `--help`
output.

## `pkg/derivation`

A parser for Nix `.drv` files.
Functions to calculate derivation paths and output hashes.

## `pkg/derivation/store`

A Structure to hold derivation graphs.

## `pkg/hash`

Methods to serialize and deserialize some of the hashes used in nix code and
`.narinfo` files.

## `pkg/nar`

A Nix ARchive (NAR) file Reader and Writer, with an interface similar to
`archive/tar` from the stdlib, as well as a `DumpPath` method, which
will assemble a NAR representation of a local file system path.

## `pkg/nar/ls`

A parser for .ls files (providing an index for .nar files)

## `pkg/nar/narinfo`

A parser and generator for `.narinfo` files.

## `pkg/nixbase32`

An implementation of the slightly odd "base32" encoding that's used in Nix,
providing some of the functions in `encoding/base32.Encoding`.

## `pkg/storepath`

A parser and regexes for Nix Store Paths.

## `pkg/storepath/references`

A Nix Store path reference scanner.

## `pkg/sqlite`

A collection of interfaces and utilities for writing to and querying various `sqlite` databases that Nix uses.

[sqlc]: https://github.com/sqlc-dev/sqlc

## `pkg/sqlite/binary_cache_v6`

[SQLC] generated code for querying the Nar Info Disk Cache, typically located at `$XDG_CACHE_HOME/nix/binary-cache-v6.sqlite`.

## `pkg/sqlite/eval_cache_v5`

[SQLC] generated code for querying an instance of the Eval Cache, typically located at `$XDG_CACHE_HOME/nix/eval-cache-v5/*.sqlite`.

## `pkg/sqlite/fetcher_cache_v2`

[SQLC] generated code for querying the fetcher cache, typically located in `$XDG_CACHE_HOME/nix/fetcher-cache-v2.sqlite`.

## `pkg/sqlite/nix_v10`

[SQLC] generated code for querying the main Nix database, typically located in `/nix/var/nix/db.sqlite`.

## `pkg/wire`

Methods to parse and produce fields used in the low-level Nix wire protocol.
