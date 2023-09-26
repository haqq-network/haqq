{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/23.05";

    flake-utils.url = "github:numtide/flake-utils";
    flake-compat = {
      url = "github:edolstra/flake-compat";
      flake = false;
    };
  };

  outputs = { self, nixpkgs, flake-utils, ... }:

    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        rev = self.rev or self.dirtyRev;
      in
      {
        packages =
          rec {
            haqqd = pkgs.callPackage ./nix/package.nix {
              rev = rev;
              # FIXME: figure out why test fail in nix
              doCheck = true;
            };

            haqqdNoCheck = pkgs.callPackage ./nix/package.nix {
              rev = rev;
              doCheck = false;
            };

            default = haqqdNoCheck;
          };

        devShell = pkgs.mkShell
          {
            buildInputs =
              with pkgs;
              [
                gh
                yarn
                go
              ];
          };
      }
    );
}
