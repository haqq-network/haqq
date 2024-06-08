{ ... }:
{
  system.stateVersion = "23.11";

  /*
    sops.age.sshKeyPaths = [ "/etc/ssh/ssh_host_ed25519_key" ];
      sops.secrets.grafana_secret_key = {
      mode = "0400";
      owner = config.users.users.grafana-agent.name;
      sopsFile = config.services.haqqd.grafana.sopsFile;
      restartUnits = [ "grafana-agent-flow.service" ];
      };
      */

  services.haqqd-supervised = {
    enable = true;
    deleteOldBackups = 7;
    config = {
      statesync = {
        enable = true;

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
    openFirewall = true;

    # not testing grafana-agent here
    grafana.enable = false;
  };
}
