#!/bin/sh

# This (re-)builds a bunch of fixture files from this folder.

# It requires the following binaries to be in $PATH:
# - nix-instantiate

# /nix/store/0hm2f1psjpcwg8fijsmr4wwxrx59s092-bar.drv
bar=$(nix-instantiate derivation_sha256.nix -A bar)
cp $bar .

# /nix/store/4wvvbi4jwn0prsdxb7vs673qa5h9gr7x-foo.drv
foo=$(nix-instantiate derivation_sha256.nix -A foo)
cp $foo .
