{ pkgs, haqqdPackage }:
let
  inherit (pkgs) stdenv callPackage;
in
stdenv.mkDerivation {
  name = "haqqd-default-config";
  version = haqqdPackage.version;

  buildInputs = [ haqqdPackage ];

  dontUnpack = true;

  CHAIN_ID = "haqqd_11235-1";

  buildPhase = ''
    haqqd init "haqqd-node" --chain-id $CHAIN_ID --home .

    mkdir $out
    mv config/app.toml $out
    mv config/config.toml $out
  '';
}
