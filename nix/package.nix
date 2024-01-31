{ pkgs, nix-filter }:
let
  version = "1.7.1";
  inherit (pkgs) lib buildGo120Module;
in
buildGo120Module {
  name = "haqqd";
  inherit version;

  # only rebuild if package-related files are changed
  src = nix-filter.filter {
    root = ../.;

    exclude = [
      "nix/"
      "flake.nix"
      "flake.lock"
    ];
  };

  # if new version is released, set this to lib.fakeSha256, run
  # nix build .#haqqd
  # and see acual hash in the error message
  vendorHash = "sha256-LBN+o0XVqF8GGPNwIXi9sYrFwcGGp6BFGMiroSji4hE=";

  # https://ryantm.github.io/nixpkgs/languages-frameworks/go/
  proxyVendor = true;

  # if some tests are intentionally failing, uncomment this
  # doCheck = false;

  preBuild = ''
    export HOME=$TMP
  '';

  meta.platforms = lib.platforms.unix ++ lib.platforms.windows;
}
