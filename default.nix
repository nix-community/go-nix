{ system ? builtins.currentSystem
, inputs ? import ./flake.lock.nix { }
, devshell ? import inputs.devshell { }
}:
let
  nixpkgs = import inputs.nixpkgs {
    inherit system;
    config = { };
    overlays = [ ];
  };
in
{
  devShell = devshell.mkShell {
    packages = with nixpkgs; [
      nixpkgs-fmt
      golangci-lint
      gofumpt
      go
      gcc
    ];
    commands = with nixpkgs; [
      {
        name = "fmt";
        help = "Format code";
        command = ''
          ${treefmt}/bin/treefmt
        '';
      }
      {
        name = "lint";
        help = "Lint code";
        command = ''
          ${golangci-lint}/bin/golangci-lint run
        '';
      }
      {
        name = "tests";
        help = "Run unitests";
        command = ''
          ${go}/bin/go test -v ./...
        '';
      }
    ];
  };
}
