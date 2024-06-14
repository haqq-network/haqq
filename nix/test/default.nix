{ pkgs, self, ... }:
pkgs.nixosTest {
  name = "haqqd";

  nodes.machine = _: {
    imports = [ self.nixosModules.default ];

    services.haqqd = {
      enable = true;
      settings = {
        app = {
          pruning = "custom";
          pruning-keep-recent = "1000";
          pruning-interval = "10";
          api.enable = false;
        };
      };
    };
  };

  testScript = ''
    machine.wait_for_unit("haqqd.service")
  '';
}
