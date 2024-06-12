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

  # https://nix.dev/tutorials/nixos/integration-testing-using-virtual-machines.html
  # testScript = ''
  #   machine.start()

  #   machine.wait_for_file("/var/lib/haqqd/.haqqd/.bootstrapped")

  #   machine.wait_for_open_port(26656)

  #   timeout = 60 * 5 # 5 minutes
  #   text = "Applied snapshot chunk"

  #   machine.wait_until_succeeds(f"journalctl -u haqqd.service --since -1m --grep='{text}'", timeout= timeout)

  #   timeout = ${toString globalTimeout}
  #   text = "commit synced commit"

  #   machine.wait_until_succeeds(f"journalctl -u haqqd.service --since -1m --grep='{text}'", timeout = timeout)
  # '';
}
