{
  buildGoApplication,
  gitignore,
  go,
  haqq,
  haqq-module-test,
  lib,
  rev,
  testers,
}:
buildGoApplication rec {
  pname = "haqq";
  version = import ./version.nix;

  src = lib.cleanSourceWith rec {
    name = "${pname}-source";
    src = ../.;
    filter = gitignore.lib.gitignoreFilterWith {
      basePath = src;
      extraRules = ''
        .github/
        CODEOWNERS
        CODE_OF_CONDUCT.md
        README.md
        nix/
        flake.lock
        *.nix
      '';
    };
  };

  inherit go;

  modules = ../gomod2nix.toml;

  subPackages = [ "cmd/haqqd" ];

  CGO_ENABLED = "1";

  ldflags = [
    "-X github.com/cosmos/cosmos-sdk/version.Name=haqq"
    "-X github.com/cosmos/cosmos-sdk/version.AppName=haqqd"
    "-X github.com/cosmos/cosmos-sdk/version.Version=${version}"
    "-X github.com/cosmos/cosmos-sdk/version.Commit=${rev}"
    "-X github.com/cosmos/cosmos-sdk/version.BuildTags=${lib.concatStringsSep "," tags}"
  ];

  tags = [
    "ledger"
    "netgo"
  ];

  doCheck = false;
  preCheck = ''
    # Some tests require the HOME environment variables to be set.
    export HOME="$(mktemp -d)"
  '';

  postInstall = ''
    # Install pre-generated configuration options.
    $out/bin/haqqd init default --home . --chain-id haqq_54211-3
    install -Dm644 -t $out/share/haqqd/config config/app.toml
    install -Dm644 -t $out/share/haqqd/config config/client.toml
    install -Dm644 -t $out/share/haqqd/config config/config.toml
  '';

  passthru.tests = {
    version = testers.testVersion { package = haqq; };
    haqq = haqq-module-test;
  };

  meta = with lib; {
    description = "Shariah-compliant Web3 platform";
    longDescription = ''
      Haqq is a scalable, high-throughput Proof-of-Stake blockchain that is
      fully compatible and interoperable with Ethereum. It's built using the
      Cosmos SDK which runs on top of CometBFT consensus engine. Ethereum
      compatibility allows developers to build applications on Haqq using the
      existing Ethereum codebase and toolset, without rewriting smart contracts
      that already work on Ethereum or other Ethereum-compatible networks.
      Ethereum compatibility is done using modules built by Tharsis for their
      Evmos network.
    '';
    homepage = "https://haqq.network";
    license = licenses.asl20;
    mainProgram = "haqqd";
  };
}
