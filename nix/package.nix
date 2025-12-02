{ rev
, nix-gitignore
, buildGoApplication
, go
, lib
, stdenv
}:
let
  name = "haqq";
  pname = "${name}";
  version =
    (import ./version.nix { });
  tags = [ "ledger" "netgo" ];
  ldflags = [
    "-X github.com/cosmos/cosmos-sdk/version.Name=evmos"
    "-X github.com/cosmos/cosmos-sdk/version.AppName=${pname}"
    "-X github.com/cosmos/cosmos-sdk/version.Version=${version}"
    "-X github.com/cosmos/cosmos-sdk/version.BuildTags=${lib.concatStringsSep "," tags}"
    "-X github.com/cosmos/cosmos-sdk/version.Commit=${rev}"
    # "-X github.com/cosmos/cosmos-sdk/types.DBBackend=${dbBackend}"
  ];
in
buildGoApplication rec {
  inherit name version go ldflags;

  modules = ../gomod2nix.toml;
  CGO_ENABLED = "1";

  # prevent rebuilds on irrelevant files changes
  # https://ryantm.github.io/nixpkgs/functions/nix-gitignore/
  src = nix-gitignore.gitignoreSource [
    ".github/"
    "nix/"
    "*.nix"
    "flake.lock"
  ] ../.;

  pwd = src;

  subPackages = [ "cmd/haqqd" ];

  doCheck = false;

  # tests require writeable $HOME
  preCheck = ''
    export HOME=$(mktemp -d)
  '';

  # Fix RPATH to remove forbidden /build/ references
  # Process all ELF binaries in the output to remove problematic RPATHs
  postFixup = lib.optionalString stdenv.isLinux ''
    for binary in $out/bin/*; do
      [ -f "$binary" ] || continue
      patchelf --remove-rpath "$binary" 2>/dev/null || patchelf --set-rpath "" "$binary" 2>/dev/null || true
    done
  '';
}