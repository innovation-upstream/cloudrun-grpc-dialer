{ pkgs ? import <nixpkgs> {}, ... }:

pkgs.mkShell {
  buildInputs = with pkgs;
    [
    bazel_6
    go_1_19
    jdk11
  ];
}

