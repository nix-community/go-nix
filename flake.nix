{
  description = "go-nix - Nix experiments written in go";

  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.devshell.url = "github:numtide/devshell";

  outputs =
    { self, nixpkgs, flake-utils, devshell }:
    {
      overlay = import ./overlay.nix;
    }
    //
    (
      flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            devshell.overlay
            self.overlay
          ];
        };
      in
      {
        packages = pkgs.go-nix;
        devShell = pkgs.mkDevShell.fromTOML ./devshell.toml;
        # devShell = pkgs.mkShell {
        #   buildInputs = [ pkgs.go ];
        # };
      }
      )
    );
}
