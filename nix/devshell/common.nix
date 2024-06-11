{ pkgs, go, ... }:
{
  packages = with pkgs; [
    (gomod2nix.override { inherit go; })
    buf
    clang-tools
    codespell
    go
    golangci-lint
    grpc-gateway
    protobuf
  ];
}
