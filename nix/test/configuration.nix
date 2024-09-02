{ pkgs, config, lib, ... }:
{
  system.stateVersion = "24.05";

  /*
    sops.age.sshKeyPaths = [ "/etc/ssh/ssh_host_ed25519_key" ];
      sops.secrets.grafana_secret_key = {
      mode = "0400";
      owner = config.users.users.grafana-agent.name;
      sopsFile = config.services.haqqd.grafana.sopsFile;
      restartUnits = [ "grafana-agent-flow.service" ];
      };
      */

  networking.firewall = {
    enable = true;
    allowedTCPPorts = [
      26656 # tendermint rpc
    ];
  };

  services.haqqd-supervised = {
    enable = true;
    deleteOldBackups = 7;
    config = {
      statesync = {
        enable = true;

        # get block hash from ping pub
        # https://ping.pub/haqq/block/<current - 1k>
        trust_height = 9623214;
        trust_hash = "6E784CF9689F635DF7521B77A737E4BD7048699A93442C5E1E926B4A2736C83A";

        rpc_servers = "https://rpc.tm.haqq.network:443,https://m-s1-tm.haqq.sh:443";
      };

      # small chance to get discovered in a little time of running the test
      # so we are increasing outbound peers for better connectivity
      p2p.max_num_outbound_peers = 40;
    };
    app = {
      pruning = "custom";
      pruning-keep-recent = "1000";
      pruning-interval = "10";

      api.enable = false;
      telemetry.prometheus-retention-time = 3600;
    };

    # not testing grafana-agent here
    grafana.enable = false;
  };
}
