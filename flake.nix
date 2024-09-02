{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2405.*.tar.gz";
    nixpkgs-unstable.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-utils.url = "github:numtide/flake-utils";

    devenv.url = "github:cachix/devenv";
    devenv.inputs.nixpkgs.follows = "nixpkgs-unstable";

    gomod2nix.url = "github:nix-community/gomod2nix/master";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs-unstable";
    gomod2nix.inputs.flake-utils.follows = "flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-unstable,
      flake-utils,
      devenv,
      gomod2nix,
      ...
    }@inputs:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        overlays = [ gomod2nix.overlays.default ];
        pkgs = import nixpkgs { inherit system overlays; };
        pkgsUnstable = import nixpkgs-unstable { inherit system overlays; };

        # match go x.x in go.mod
        gomod = builtins.readFile ./go.mod;
        goVersion = builtins.match ".*[\n]go ([[:digit:]]*)\.([[:digit:]]*)[\.]*([[:digit:]]*)[\n].*" gomod;

        go = pkgs."go_${builtins.head goVersion}_${builtins.elemAt goVersion 1}";
      in
      {
        packages = rec {
          nixos-test = pkgs.callPackage ./nix/test { overlay = self.overlays.default; };
          haqq = pkgsUnstable.callPackage ./nix/package.nix {
            inherit (pkgsUnstable) buildGoApplication;
            inherit go;
            rev = if (self ? rev) then self.rev else self.dirtyRev;
          };
          haqq-with-tests = haqq.overrideAttrs (_: {
            subPackages = null;
            doCheck = true;
          });
          default = haqq;
        };

        devShells = {
          default = devenv.lib.mkShell {
            inherit inputs pkgs;
            modules = [
              (import ./nix/devshell/common.nix { inherit pkgs pkgsUnstable go; })
              (import ./nix/devshell { inherit pkgs pkgsUnstable; })
            ];
          };

          ci = devenv.lib.mkShell {
            inherit inputs pkgs;
            modules = [
              (import ./nix/devshell/common.nix { inherit pkgs pkgsUnstable go; })
              (import ./nix/devshell/ci.nix { inherit pkgs pkgsUnstable go; })
            ];
          };
        };
      }
    )
    // {

      overlays.default = prev: final: {
        inherit (inputs.cosmos.packages.${prev.system}) cosmovisor;
        inherit (self.packages.${prev.system}) haqq;
        grafana-agent-unstable = inputs.nixpkgs-unstable.legacyPackages.${prev.system}.grafana-agent;
      };

      nixosModules = {
        haqqdSupervised = {
          imports = [ ./nix/nixos-module ];

          nixpkgs.overlays = [ self.overlays.default ];
        };
      };

      nixConfig = {
        extra-trusted-public-keys = [
          "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
          "haqq.cachix.org-1:m8QJypf2boIKRBz4BvVyGPo7gHQoj4D6iMGCmGozNEg="
        ];
        extra-substituters = [
          "https://devenv.cachix.org"
          "https://haqq.cachix.org"
        ];
      };
    };
}
