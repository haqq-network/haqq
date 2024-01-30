{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2311.*.tar.gz";
    nixpkgs-unstable.url = "github:NixOS/nixpkgs/nixos-unstable";

    nix-filter.url = "github:numtide/nix-filter";

    cosmos.url = "https://flakehub.com/f/informalsystems/cosmos.nix/0.*.tar.gz";
    cosmos.inputs.nixpkgs.follows = "nixpkgs-unstable";

    flake-utils.url = "github:numtide/flake-utils";

    devenv.url = "github:cachix/devenv";
    devenv.inputs.nixpkgs.follows = "nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, flake-utils, devenv, nix-filter, ... }@inputs:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          pkgsUnstable = nixpkgs-unstable.legacyPackages.${system};
        in
        {

          packages = {
            nixos-test = pkgs.callPackage ./nix/test {
              overlay = self.overlays.default;
            };
            haqqd = pkgs.callPackage ./nix/package.nix {
              nix-filter = nix-filter.lib;
            };
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
        inherit (self.packages.${prev.system}) haqqd;
        grafana-agent-unstable = inputs.nixpkgs-unstable.legacyPackages.${prev.system}.grafana-agent;
      };

      nixosModules = {
        haqqdSupervised = {
          imports = [ ./nix/nixos-module ];

          nixpkgs.overlays = [ self.overlays.default ];
        };
      };

      nixConfig = {
        extra-trusted-public-keys = "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=";
        extra-substituters = "https://devenv.cachix.org";
      };
    };
}

