rec {
  bar = builtins.derivation {
    name = "bar";
    builder = ":";
    system = ":";
    outputHash = "08813cbee9903c62be4c5027726a418a300da4500b2d369d3af9286f4815ceba";
    outputHashAlgo = "sha256";
    outputHashMode = "recursive";
  };

  foo = builtins.derivation {
    name = "foo";
    builder = ":";
    system = ":";
    inherit bar;
  };
}
