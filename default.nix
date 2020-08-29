{ pkgs }:

pkgs.buildGoModule {
  pname = "go-nix";
  version = "latest";
  src = pkgs.lib.cleanSource ./.;
  vendorSha256 = "0knmyqls62i9y8h6h8p1h9m768ni58anfdiw5ljwc3n40l20hk2z";
}
