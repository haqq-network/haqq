{ pkgs, ... }:
{
  dotenv.enable = true;

  packages = with pkgs; [
    act
    dasel
    gh
    jq
    yq
  ];

  enterShell = ''
    export PATH=node_modules/.bin:$PATH
  '';

  pre-commit.hooks = {
    nil.enable = true;
    deadnix.enable = true;
    statix.enable = true;
    nixfmt = {
      enable = true;
      package = pkgs.nixfmt-rfc-style;
    };

    golangci-lint.enable = false; # FIXME Fails!
    gomod2nix-generate = {
      enable = true;
      name = "gomod2nix-generate";
      always_run = true;
      entry = "${pkgs.gomod2nix}/bin/gomod2nix generate";
      pass_filenames = false;
    };
  };
}
