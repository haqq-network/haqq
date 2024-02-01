{ rev, nix-filter, mkCosmosGoApp, lib }:
mkCosmosGoApp {
  name = "haqq";
  version = "v1.7.1";

  goVersion = "1.20";
  tags = [ "netgo" ];
  engine = "cometbft/cometbft";

  # if new version is released, set this to lib.fakeHash, run
  # nix build .#haqqd
  # and see acual hash in the error message
  vendorHash = "sha256-LBN+o0XVqF8GGPNwIXi9sYrFwcGGp6BFGMiroSji4hE=";

  proxyVendor = true;

  inherit rev;
  src = nix-filter.filter {
    root = ../.;

    exclude = [
      ".github/"
      "nix/"
      "flake.nix"
      "flake.lock"
    ];
  };
}
