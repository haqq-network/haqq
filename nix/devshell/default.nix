{ pkgs, pkgsUnstable, ... }:
{
  dotenv.enable = true;

  packages = with pkgs;
    [
      pkgsUnstable.act
      gh

      yarn
      nodejs

      jq
      yq

      statik

      golangci-lint
    ];

  enterShell = ''
    export PATH=node_modules/.bin:$PATH
  '';

  pre-commit.hooks = {
    golangci-lint.enable = true;

    gomod2nix-generate = {
      enable = true;
      name = "gomod2nix-generate";
      always_run = true;
      entry = "gomod2nix generate";
      pass_filenames = false;
    };
  };
}
