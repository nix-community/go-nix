builtins.derivation {
  name = "nested-json";
  builder = ":";
  system = ":";
  json = builtins.toJSON {
    hello = "moto\n";
  };
}
