{ pkgs, pkgsUnstable, go, ... }:
{
  packages = with pkgs;
    [
      protobuf
      buf
      clang-tools

      go
      (pkgsUnstable.gomod2nix.override {
        inherit go;
      })
      golangci-lint

      (callPackage ../grpc-gateway.nix {
        inherit pkgs;
      })
    ];
}
