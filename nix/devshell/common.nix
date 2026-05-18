{ pkgs, pkgsUnstable, go, ... }:
{
  packages = with pkgs;
    [
      protobuf
      buf
      clang-tools
      codespell
      jq
      git
      golangci-lint
      go
      (pkgsUnstable.gomod2nix.override {
        inherit go;
      })

      (callPackage ../grpc-gateway.nix {
        inherit pkgs;
      })
    ];
  
  # Install statik via go install since it's not available as a nix package
  enterShell = ''
    export GOPATH="''${GOPATH:-$HOME/go}"
    export PATH="$PATH:$GOPATH/bin"
    if ! command -v statik &> /dev/null; then
      echo "Installing statik..."
      (cd /tmp && go install github.com/rakyll/statik@v0.1.6)
    fi
  '';
}
