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

  # Prevent RPATH from being set to /build/ during CGO linking
  # Use linker flags appropriate for each platform to prevent RPATH issues
  CGO_LDFLAGS = 
    if stdenv.isLinux then "-Wl,--disable-new-dtags"
    else if stdenv.isDarwin then "-headerpad_max_install_names"
    else "";

  # Fix RPATH in preFixup (runs before fixupPhase which checks for forbidden references)
  # Cross-platform: gracefully handle RPATH removal on platforms that support it
  preFixup = ''
    for binary in $out/bin/*; do
      [ -f "$binary" ] || continue
      # Remove RPATH if patchelf is available (Linux/ELF platforms)
      if command -v patchelf >/dev/null 2>&1; then
        patchelf --remove-rpath "$binary" 2>/dev/null || patchelf --set-rpath "" "$binary" 2>/dev/null || true
      fi
    done
  '';
}