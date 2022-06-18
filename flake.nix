{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/22.05;
    nixpkgs-unstable.url = github:NixOS/nixpkgs/nixpkgs-unstable;
    utils.url = github:numtide/flake-utils;
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, utils }:
    utils.lib.eachDefaultSystem (
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
            ]
          ) ++ (
            with pkgsUnstable; [
            ]
          );
        };

      }
    );
}
