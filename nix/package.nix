{ pkgs, lib, rev, doCheck, ... }:
let
  version = "1.5.0";
  name = "haqq";
  parser = import ./goModParser.nix;
  src = ../.;
  go-mod = parser (builtins.readFile "${src}/go.mod");
  tmversion = go-mod.require."github.com/tendermint/tendermint".version;
  build_tags = [ "netgo" "gcc" "ledger" ];
in
with pkgs; buildGoModule rec {
  inherit name src version doCheck;

  tags = build_tags;
  ldflags = [
    "github.com/cosmos/cosmos-sdk/version.Name=haqq"
    "github.com/cosmos/cosmos-sdk/version.AppName=haqqd"
    "github.com/cosmos/cosmos-sdk/version.Version=${version}"
    "github.com/cosmos/cosmos-sdk/version.Commit=${rev}"
    "github.com/cosmos/cosmos-sdk/version.BuildTags=${lib.concatStringsSep "," build_tags}"
    "github.com/tendermint/tendermint/version.TMCoreSemVer=${tmversion}"
  ];

  # panic: failed to read upgrade info from disk: could not create directory "/homeless-shelter/.haqqd/data": mkdir /homeless-shelter: read-only file system [recovered]
  preCheck = ''
    # mkdir -p $out/bin
    export HOME=$TMP
  '';

  proxyVendor = true;
  vendorSha256 = "sha256-xUpuqtgBGSf8M5QmJ8+KhKm7bz+S6qByL74ciyLZInY=";

  meta.platforms = lib.platforms.unix;
}
