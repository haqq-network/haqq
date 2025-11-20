{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    nixpkgs-unstable.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-utils.url = "github:numtide/flake-utils";

    devenv.url = "github:cachix/devenv";
    devenv.inputs.nixpkgs.follows = "nixpkgs-unstable";

    git-hooks.url = "github:cachix/git-hooks.nix";
    git-hooks.inputs.nixpkgs.follows = "nixpkgs-unstable";
    devenv.inputs.git-hooks.follows = "git-hooks";

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
        goMajor = builtins.head goVersion;
        goMinor = builtins.elemAt goVersion 1;
        
        # Try to get the Go version from pkgsUnstable first (since we use it for building),
        # then fallback to pkgs, then to latest available version
        # Go 1.24 might not be available in nixpkgs yet, so we fallback to 1.23
        go = if pkgsUnstable ? "go_${goMajor}_${goMinor}" then
          pkgsUnstable."go_${goMajor}_${goMinor}"
        else if pkgs ? "go_${goMajor}_${goMinor}" then
          pkgs."go_${goMajor}_${goMinor}"
        else if pkgsUnstable ? "go_1_24" then
          pkgsUnstable.go_1_24
        else if pkgs ? "go_1_24" then
          pkgs.go_1_24
        else if pkgsUnstable ? "go_1_23" then
          pkgsUnstable.go_1_23
        else if pkgs ? "go_1_23" then
          pkgs.go_1_23
        else
          pkgs.go;
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

      overlays.default = prev: final:
        let
          # Get system from the package set  
          system = prev.stdenv.hostPlatform.system;
          # Get grafana-agent from nixpkgs-unstable
          grafanaAgentPkg = (import inputs.nixpkgs-unstable {
            inherit system;
            overlays = [ ];
          }).grafana-agent;
        in
        {
          cosmovisor = prev.callPackage ./nix/cosmovisor.nix { };
          grafana-agent-unstable = grafanaAgentPkg;
        };

      nixosModules = {
        haqqdSupervised = {
          imports = [ ./nix/nixos-module ];

          nixpkgs.overlays = [
            self.overlays.default
            (final: prev: {
              haqq = self.packages.${prev.stdenv.hostPlatform.system}.haqq;
            })
          ];
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
