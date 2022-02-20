{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    devshell.url = "github:numtide/devshell";
  };
  outputs = { self, nixpkgs, flake-utils, devshell }@inputs:
    flake-utils.lib.eachSystem [ "x86_64-linux" ] (system:
      let
        pkgs = import ./. {
          inherit system inputs;
        };
      in
      {
        devShell = pkgs.devShell;
        checks = {
          fmt = with nixpkgs.legacyPackages."${system}"; runCommandLocal "fmt"
            {
              buildInputs = [ gofumpt nixpkgs-fmt rsync ];
            } ''
            export HOME=$TMP
            cd $TMP
            rsync --chmod=D0770,F0660 -a ${./.}/ ./
            ${treefmt}/bin/treefmt --fail-on-change
            touch $out
          '';
          lint = with nixpkgs.legacyPackages."${system}"; runCommandLocal "lint"
            {
              buildInputs = [ go gcc rsync ];
            } ''
            export GOLANGCI_LINT_CACHE=$TMPDIR/.cache/golangci-lint
            export GOCACHE=$TMPDIR/.cache/go-build
            export GOMODCACHE="$TMPDIR/.cache/mod"
            mkdir $out
            cd $out
            rsync -a ${./.}/ ./
            ${golangci-lint}/bin/golangci-lint run
            touch $out
          '';
        };
      });
}
