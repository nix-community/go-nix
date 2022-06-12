rec {
  bar = builtins.derivation {
    name = "bar";
    builder = ":";
    system = ":";
    outputHash = "0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33";
    outputHashAlgo = "sha1";
    outputHashMode = "recursive";
  };

  foo = builtins.derivation {
    name = "foo";
    builder = ":";
    system = ":";
    inherit bar;
  };
}
