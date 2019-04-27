with import <nixpkgs> {};
mkShell {
  buildInputs = [ go ];
  shellHook = ''
    unset GOPATH GOROOT
    export GO111MODULE=on
  '';
}
