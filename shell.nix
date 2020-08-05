with import <nixpkgs> {};
mkShell {
  buildInputs = [ go awscli minio minio-client];
  shellHook = ''
    unset GOPATH GOROOT
    export GO111MODULE=on
  '';
}
