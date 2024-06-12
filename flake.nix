{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/release-24.05";
    cosmos.url = "github:informalsystems/cosmos.nix";
    devenv.url = "github:cachix/devenv";
    gomod2nix.url = "github:nix-community/gomod2nix";
    gitignore.url = "github:hercules-ci/gitignore.nix";
    flake-utils.url = "github:numtide/flake-utils";
    flake-compat.url = "github:edolstra/flake-compat";
  };

  nixConfig = {
    extra-trusted-public-keys = [
      "cosmosnix.store-1:O28HneR1MPtgY3WYruWFuXCimRPwY7em5s0iynkQxdk="
      "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
      "haqq.cachix.org-1:m8QJypf2boIKRBz4BvVyGPo7gHQoj4D6iMGCmGozNEg="
      "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
    ];
    extra-substituters = [
      "https://cosmos-nix.cachix.org"
      "https://devenv.cachix.org"
      "https://haqq.cachix.org"
      "https://nix-community.cachix.org"
    ];
  };

  outputs =
    {
      cosmos,
      devenv,
      flake-utils,
      gomod2nix,
      nixpkgs,
      self,
      gitignore,
      ...
    }@inputs:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlays.default ];
        };

        # Match Go version in the go.mod file.
        goMod = builtins.readFile ./go.mod;
        goVersion = builtins.match ".*[\n]go ([[:digit:]]*)\.([[:digit:]]*)[\n].*" goMod;
        go = pkgs."go_${builtins.head goVersion}_${builtins.elemAt goVersion 1}";
      in
      rec {
        apps.haqq = {
          type = "app";
          program = nixpkgs.lib.getExe packages.haqq;
        };

        packages = rec {
          default = haqq;
          haqq = pkgs.callPackage ./nix/package.nix {
            rev = if (self ? rev) then self.rev else self.dirtyRev;
            inherit (pkgs) buildGoApplication;
            inherit
              haqq
              haqq-module-test
              gitignore
              go
              ;
          };

          # For local development. Prevents recompiles on Git tree changes.
          haqq-no-rev = haqq.overrideAttrs (_: {
            rev = "norev";
          });

          # Also runs the test suite.
          haqq-with-tests = haqq.overrideAttrs (_: {
            subPackages = null;
            doCheck = true;
          });

          # NixOS module test.
          haqq-module-test = pkgs.callPackage ./nix/test { inherit self; };
        };

        devShells = with devenv.lib; {
          default = mkShell {
            inherit pkgs inputs;
            modules = [
              (import ./nix/devshell/common.nix { inherit pkgs go; })
              (import ./nix/devshell { inherit pkgs go; })
            ];
          };

          ci = mkShell {
            inherit pkgs inputs;
            modules = [
              (import ./nix/devshell/common.nix { inherit pkgs go; })
              (import ./nix/devshell/ci.nix { inherit pkgs go; })
            ];
          };
        };
      }
    )
    // {
      overlays.default = final: _prev: {
        inherit (inputs.cosmos.packages.${final.system}) cosmovisor;
        inherit (self.packages.${final.system}) haqq;
      };

      nixosModules.default = {
        imports = [ ./nix/nixos-module ];
        nixpkgs.overlays = [ self.overlays.default ];
      };
    };
}
