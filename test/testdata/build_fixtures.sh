#!/bin/sh

# This (re-)builds a bunch of fixture files from this folder.

# It requires the following binaries to be in $PATH:
# - nix-instantiate

# /nix/store/0hm2f1psjpcwg8fijsmr4wwxrx59s092-bar.drv
bar=$(nix-instantiate derivation_sha256.nix -A bar)
cp $bar .
bar_json_path=$(basename $bar).json
nix show-derivation $bar > $bar_json_path

# /nix/store/4wvvbi4jwn0prsdxb7vs673qa5h9gr7x-foo.drv
foo=$(nix-instantiate derivation_sha256.nix -A foo)
cp $foo .
foo_json_path=$(basename $foo).json
nix show-derivation $foo > $foo_json_path

# /nix/store/ss2p4wmxijn652haqyd7dckxwl4c7hxx-bar.drv
bar=$(nix-instantiate derivation_sha1.nix -A bar)
cp $bar .
bar_json_path=$(basename $bar).json
nix show-derivation $bar > $bar_json_path

# /nix/store/ch49594n9avinrf8ip0aslidkc4lxkqv-foo.drv
foo=$(nix-instantiate derivation_sha1.nix -A foo)
cp $foo .
foo_json_path=$(basename $foo).json
nix show-derivation $foo > $foo_json_path

# /nix/store/h32dahq0bx5rp1krcdx3a53asj21jvhk-has-multi-out.drv
multi_out=$(nix-instantiate derivation_multi-outputs.nix)
cp $multi_out .
multi_out_json_path=$(basename $multi_out).json
nix show-derivation $multi_out > $multi_out_json_path
