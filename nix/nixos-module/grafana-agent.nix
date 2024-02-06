{ pkgs, config, lib, ... }:
let
  haqqCfg = config.services.haqqd-supervised;
  cfg = haqqCfg.grafana;
in
{
  config = lib.mkIf (haqqCfg.enable && cfg.enable) {

    # This is using an age key that is expected to already be in the filesystem
    # sops.age.keyFile = "/var/lib/sops-nix/key.txt";
    # This will generate a new key if the key specified above does not exist
    # sops.age.generateKey = true;

    users.users.grafana-agent = {
      isSystemUser = true;
      createHome = true;
      home = "/var/lib/grafana-agent-flow";
      group = config.users.groups.grafana-agent.name;
    };

    users.groups.grafana-agent = { };

    systemd.services.grafana-agent-flow = {
      wantedBy = [ "multi-user.target" ];
      environment = {
        AGENT_MODE = "flow";
        GRAFANA_METRICS_URL = cfg.metricsUrl;
        GRAFANA_LOGS_URL = cfg.logsUrl;
        GRAFANA_SECRET_KEY = cfg.secretKeyPath;
      };
      serviceConfig =
        let
          configFile = pkgs.substituteAll {
            src = ./grafana-agent.river;
            instance = cfg.instance;
          };
        in
        {
          ExecStart = "${lib.getExe cfg.package} run ${configFile} --storage.path ${config.users.users.grafana-agent.home}";
          User = config.users.users.grafana-agent.name;
          Restart = "always";
          DynamicUser = true;
          RestartSec = 2;
          StateDirectory = "grafana-agent-flow";
          Type = "simple";
        };
    };
  };
}
