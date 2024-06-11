{
  pkgs ? import <nixpkgs> { },
  platform ? pkgs.stdenvNoCC.hostPlatform.system,
  version,
  ...
}:

# Not being used right now, keeping it for future reference

# This derivation downloads the haqqd binary from GitHub releases
# It's useful when you need to get a version built without nix
# on nixos it's required to set `programs.nix-ld.enable = true;`
# and `environ,ent.NIX_LD = config.environment.variables.NIX_LD`, if you run this binary in a systemd service

let
  inherit (pkgs) lib stdenv fetchurl;
  hashesFile = fetchurl {
    url = "https://github.com/haqq-network/haqq/releases/download/v${version}/checksums.txt";
    sha256 = "sha256-DcdLvirs6ecP3GK25K+ws7ikvlU9fBhToTglZxKStLw=";
  };
  hashesLines = lib.lists.remove "" (lib.strings.splitString "\n" (builtins.readFile hashesFile));

  # Function to parse a line to an attribute set
  parseLine =
    line:
    let
      parts = builtins.split "[ \t]+" line;

      hash = builtins.head parts;
      fileName = builtins.elemAt parts (builtins.length parts - 1);

      fileParts = builtins.split "_" fileName;
      os = builtins.elemAt fileParts 4;
      arch = if (builtins.elemAt fileParts 6) == "x86" then "x86_64" else "arm64";
      name =
        builtins.replaceStrings
          [
            "Darwin"
            "Linux"
            "Windows"
            "arm64"
          ]
          [
            "darwin"
            "linux"
            "windows"
            "aarch64"
          ]
          "${arch}-${os}";
    in
    {
      inherit name;
      value = {
        inherit arch os hash;
      };
    };
  dists = builtins.listToAttrs (builtins.map parseLine hashesLines);
  dist = dists.${platform} or (throw "Unsupported platform ${platform}");
in
stdenv.mkDerivation {
  name = "haqqd-pure";
  inherit version;

  src = fetchurl {
    url = "https://github.com/haqq-network/haqq/releases/download/v${version}/haqq_${version}_${dist.os}_${dist.arch}.tar.gz";
    sha256 = dist.hash;
  };

  installPhase = ''
    echo 1
    mkdir -p $out/bin
    cp haqqd $out/bin/haqqd

    export chain_id=haqq_11235-1
    export haqqd_share=$out/share/haqqd/init
    mkdir -p $haqqd_share

    $out/bin/haqqd-patched config chain-id $chain_id --home $haqqd_share
    $out/bin/haqqd-patched init "haqq-node" --chain-id $chain_id --home $haqqd_share
  '';

  meta.platforms = lib.platforms.unix ++ lib.platforms.windows;
}
