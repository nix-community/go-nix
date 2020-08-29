{
  description = "go-nix - Nix experiments written in go";

  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.devshell.url = "github:numtide/devshell";

  outputs =
    { self, nixpkgs, flake-utils, devshell }:
    {
      overlay = import ./overlay.nix { };
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
        rec {
          packages.go-nix = pkgs.go-nix.lib;
		  defaultPackage = pkgs.go-nix.lib;
		  apps.go-nix = flake-utils.lib.mkApp { drv = pkgs.go-nix.lib; };
		  defaultApp = apps.go-nix.lib;
		  devShell = pkgs.mkDevShell.fromTOML ./devshell.toml;

          # Additional checks on top of the packages
          checks = { };
        }
      )
    );
}
