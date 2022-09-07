let
  sources = (import ./sources.nix);
  pkgs = import sources.nixpkgs { };

  profileEnv = pkgs.writeTextFile {
    name = "profile-env";
    destination = "/.profile";
    # This gets sourced by direnv. Set NIX_PATH, so `nix-shell` uses the same nixpkgs as here.
    text = ''
      export NIX_PATH=nixpkgs=${toString pkgs.path}
    '';
  };
  env = pkgs.buildEnv {
    name = "dev-env";
    paths = [
      profileEnv

      pkgs.niv

      pkgs.just

      pkgs.buf
      pkgs.go_1_19
      pkgs.protobuf
      #pkgs.protoc-gen-connect-go
      pkgs.protoc-gen-go

      pkgs.gofumpt
      pkgs.golangci-lint
    ];
  };
in
{
  inherit env pkgs;
}
