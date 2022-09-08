{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/22.05;
    nixpkgs-unstable.url = github:NixOS/nixpkgs/nixpkgs-unstable;

    flake-utils.url = "github:numtide/flake-utils";
    flake-compat = {
      url = "github:edolstra/flake-compat";
      flake = false;
    };
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        pkgsUnstable = import nixpkgs-unstable { inherit system; };
      in
      {
        devShell = pkgs.mkShell {
          buildInputs = (
            with pkgs; [
              go_1_18
              gh
              gosec
            ]
          ) ++ (
            with pkgsUnstable; [
            ]
          );
        };
      }
    );
}

