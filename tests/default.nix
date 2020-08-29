{ pkgs }:

pkgs.buildGoModule {
  pname = "go-nix-test";
  version = "latest";
  src = pkgs.lib.cleanSource ./.;
  vendorSha256 = "1wihwj2rqv18vzn4kwnqwmpx03yiv2ib9yy317nwy6392zyczv8n";
}
