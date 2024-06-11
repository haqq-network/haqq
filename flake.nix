{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/release-24.05";

    flake-utils.url = "github:numtide/flake-utils";

    devenv.url = "github:cachix/devenv";
    # NOTE Do not override inputs for devenv. It uses its own nixpkgs fork.

    gomod2nix.url = "github:nix-community/gomod2nix";

    cosmos.url = "github:informalsystems/cosmos.nix";
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
      ...
    }@inputs:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        overlays = [ gomod2nix.overlays.default ];
        pkgs = import nixpkgs { inherit system overlays; };

        # match go x.x in go.mod
        gomod = builtins.readFile ./go.mod;
        goVersion = builtins.match ".*[\n]go ([[:digit:]]*)\.([[:digit:]]*)[\n].*" gomod;

        go = pkgs."go_${builtins.head goVersion}_${builtins.elemAt goVersion 1}";
      in
      rec {
        apps.haqq = {
          type = "app";
          program = "${packages.haqq}/bin/haqqd";
        };

        packages = rec {
          nixos-test = pkgs.callPackage ./nix/test { overlay = self.overlays.default; };
          haqq = pkgs.callPackage ./nix/package.nix {
            inherit (pkgs) buildGoApplication go;
            rev = if (self ? rev) then self.rev else self.dirtyRev;
          };
          # for local development, to prevent recompiles on git tree changes
          haqq-no-rev = haqq.overrideAttrs (_: {
            rev = "norev";
          });
          haqq-with-tests = haqq.overrideAttrs (_: {
            subPackages = null;
            doCheck = true;
          });
          default = haqq;
        };

        devShells =
          let
            inherit (nixpkgs) lib;
          in
          {
            default = devenv.lib.mkShell {
              inherit inputs pkgs;
              modules = [
                (import ./nix/devshell/common.nix { inherit pkgs go; })
                (import ./nix/devshell { inherit lib pkgs; })
              ];
            };

            ci = devenv.lib.mkShell {
              inherit inputs pkgs;
              modules = [
                (import ./nix/devshell/common.nix { inherit pkgs go; })
                (import ./nix/devshell/ci.nix { inherit lib pkgs go; })
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

      nixosModules = {
        haqqdSupervised = {
          imports = [ ./nix/nixos-module ];

          nixpkgs.overlays = [ self.overlays.default ];
        };
      };
    };
}
