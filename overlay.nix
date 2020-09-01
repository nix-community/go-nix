final: prev:
{
  go-nix = prev.callPackage ./. { pkgs = prev; };
}
