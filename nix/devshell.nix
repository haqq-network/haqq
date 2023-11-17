{ pkgs, ... }:
{
  dotenv.enable = true;

  packages = with pkgs;
    [
      gh
      yarn
      nodejs

      jq
      yq

      statik

      protobuf
      buf
      clang-tools

      (callPackage ./grpc-gateway.nix { inherit pkgs; })
    ];

  enterShell = ''
    export PATH=node_modules/.bin:$PATH
  '';

  languages.go =
    {
      enable = true;
      package = pkgs.go_1_19;
    };

  scripts.ci-proto.exec = ''
    set -e

    make proto-swagger-gen

    if ! git diff --exit-code; then
    echo "Directory is not clean after swagger generation"
    exit 1
    fi
  '';
}
