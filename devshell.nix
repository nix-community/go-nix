{
  perSystem,
  pkgs,
  ...
}:
pkgs.mkShell {
  env.GOROOT = "${pkgs.go}/share/go";

  packages =
    (with pkgs; [
      delve
      pprof
      go
      gotools
      golangci-lint
      lazysql
      sqlc
    ])
    ++ (with perSystem; [
      gomod2nix.default
    ]);
}
