{ pkgs, overlay, ... }:
pkgs.nixosTest rec {
  name = "service-test";
  enableOCR = true;
  globalTimeout = 60 * 60 * 6; # hours (statesync is very slow until we migrate to iavlv1)

  nodes.machine = _: {
    virtualisation = {
      memorySize = 16 * 1024;
      diskSize = 50 * 1024;
      cores = 8;

      # https://wiki.qemu.org/Documentation/9psetup#Performance_Considerations
      # from console output:
      # 9pnet: Limiting 'msize' to 512000 as this is the maximum supported by transport virtio
      # so setting it to maximum allowed
      msize = 512000;

      graphics = false;

      forwardPorts = [
        # expose tendermint p2p on host to make it discoverable
        {
          from = "host";
          proto = "tcp";

          guest.port = 26656;
          host.port = 26656;
        }

        # ssh into the test machine to debug it
        {
          from = "host";
          proto = "tcp";

          guest.port = 22;
          host.port = 2222;
          host.address = "127.0.0.1";
        }
      ];
    };

    imports = [ ../nixos-module ];

    nixpkgs.overlays = [ overlay ];

    services.haqqd = {
      enable = true;
      settings = {
        config = {
          statesync = {
            enable = true;
            rpc_servers = "https://rpc.tm.haqq.network:443,https://m-s1-tm.haqq.sh:443";
          };
          p2p.max_num_outbound_peers = 40;
        };
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
