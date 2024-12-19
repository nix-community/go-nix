{ __findFile }:

<third_party/buildGo>.package {
  path = "src.lunatics.tech/nixsto.re/nar";

  srcs = [
    ./copy.go
    ./reader.go
    ./writer.go
  ];
}
