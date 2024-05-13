{ pkgs, pkgsUnstable, go, ... }:
{
  packages = with pkgs;
    [
      protobuf
      buf
      clang-tools
      codespell

      go
      (pkgsUnstable.gomod2nix.override {
        inherit go;
      })

      (callPackage ../grpc-gateway.nix {
        inherit pkgs;
      })
    ];
}
