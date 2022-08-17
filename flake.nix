{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/22.05;
    nixpkgs-unstable.url = github:NixOS/nixpkgs/nixpkgs-unstable;

    flake-utils = {
      url = "github:numtide/flake-utils";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    flake-compat = {
      url = "github:edolstra/flake-compat";
      inputs.nixpkgs.follows = "nixpkgs-unstable";
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
              go_1_17
              gh
            ]
          ) ++ (
            with pkgsUnstable; [
            ]
          );
        };
      }
    );
}

