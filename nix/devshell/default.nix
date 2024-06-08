{ lib, pkgs, pkgsUnstable, ... }:
{
  dotenv.enable = true;

  packages = with pkgs;
    [
      pkgsUnstable.act
      gh
      jq
      yq
      dasel
    ];

  enterShell = ''
    export PATH=node_modules/.bin:$PATH
  '';

  pre-commit.hooks = {
    golangci-lint.enable = false; # FIXME Fails!

    gomod2nix-generate = {
      enable = true;
      name = "gomod2nix-generate";
      always_run = true;
      entry = "${lib.getExe' pkgs.gomod2nix "gomod2nix"} generate";
      pass_filenames = false;
    };
  };
}
