builtins.derivation {
  name = "escape_html";
  builder = "cat < /dev/urandom > /dev/audio";
  system = ":";
}
