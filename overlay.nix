final: prev:
{
  go-nix = {
	lib = prev.callPackage ./. { pkgs = final; };
    tests = prev.callPackage ./tests { pkgs = final; };
  };
}
