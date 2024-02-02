{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2311.*.tar.gz";
    nixpkgs-unstable.url = "github:NixOS/nixpkgs/nixos-unstable";

    cosmos.url = "https://flakehub.com/f/informalsystems/cosmos.nix/0.*.tar.gz";
    cosmos.inputs.nixpkgs.follows = "nixpkgs";

    flake-utils.url = "github:numtide/flake-utils";

    devenv.url = "github:cachix/devenv";
    devenv.inputs.nixpkgs.follows = "nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, flake-utils, devenv, cosmos, ... }@inputs:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          overlays = [
            cosmos.overlays.cosmosNixLib
          ];
          pkgs = import nixpkgs {
            inherit system overlays;
          };
          pkgsUnstable = import nixpkgs-unstable {
            inherit system overlays;
          };
        in
        {
          packages = rec {
            nixos-test = pkgs.callPackage ./nix/test {
              overlay = self.overlays.default;
            };
            haqq = pkgs.callPackage ./nix/package.nix {
              inherit (pkgs) lib nix-gitignore;
              inherit (pkgs.cosmosLib) mkCosmosGoApp;
              rev = if (self ? rev) then self.rev else self.dirtyRev;
            };
            haqqNoCheck = haqq.overrideAttrs (_: { doCheck = false; });
            default = haqq;
          };

          devShells = {
            default = devenv.lib.mkShell {
              inherit inputs pkgs;
              modules = [
                (import ./nix/devshell.nix { inherit pkgs pkgsUnstable; })
              ];
            };
          };
        }
      ) // {

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

