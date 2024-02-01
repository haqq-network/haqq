{ pkgs, lib, config, ... }:
let
  cfg = config.services.haqqd-supervised;

  defaultCfg = pkgs.callPackage ./default-config.nix {
    haqqdPackage = cfg.initialPackage;
  };

  haqqdUserName = "haqqd";
  haqqdGroupName = haqqdUserName;

  defaultCfgConfigToml = lib.recursiveUpdate (lib.importTOML "${defaultCfg}/config.toml") {
    instrumentation.prometheus = true;
    chain-id = "haqq_11235-1";
    p2p.seeds = "c45991e0098b9cacb8603caf4e1cdb7e6e5f87c0@eu.seed.haqq.network:26656,e37cb47590ba46b503269ef255873e9698244d8b@us.seed.haqq.network:26656,c593e93e1fb8be8b48d4e7bab514a227aa620bf8@as.seed.haqq.network:26656";
  };
  defaultCfgAppToml = lib.recursiveUpdate (lib.importTOML "${defaultCfg}/app.toml")
    {
      telemetry = {
        enabled = true;
      };
    };
in
{
  imports = [
    ./grafana-agent.nix
    # ./nginx.nix
  ];

  options.services.haqqd-supervised =
    {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = false;
      };

      deleteOldBackups = lib.mkOption {
        type = lib.types.int;
        default = 7;
      };

      initialPackage = lib.mkOption {
        type = lib.types.package;
        default = pkgs.haqq;
      };

      config = lib.mkOption {
        type = lib.types.attrs;
        default = { };
      };

      userHome = lib.mkOption {
        type = lib.types.str;
        default = "/var/lib/haqqd";
      };

      app = lib.mkOption
        {
          type = lib.types.attrs;
          default = { };
        };

      grafana = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = true;
        };

        package = lib.mkOption {
          type = lib.types.package;
          default = pkgs.grafana-agent-unstable;
        };

        instance = lib.mkOption {
          type = lib.types.str;
          default = "haqqd";
        };

        metricsUrl = lib.mkOption {
          type = lib.types.str;
        };

        logsUrl = lib.mkOption {
          type = lib.types.str;
        };

        secretKeyPath = lib.mkOption {
          type = lib.types.path;
        };
      };
    };

  config = lib.mkIf cfg.enable {
    # to support launching of binaries downloaded by cosmovisor from github releases
    programs.nix-ld.enable = true;

    users.users.${haqqdUserName} =
      {
        isSystemUser = true;
        home = cfg.userHome;
        createHome = true;
        group = haqqdGroupName;
      };

    users.groups.${haqqdGroupName} =
      { };


    systemd.services =
      {
        haqqd-bootstrap = {
          serviceConfig =
            let
              haqqd-init = pkgs.writeShellApplication {
                name = "haqqd-bootstrap";
                runtimeInputs = with pkgs; [
                  cfg.initialPackage
                  curl
                  gnused
                  coreutils
                  which

                  dig.dnsutils
                ];

                text = builtins.readFile ./haqqd-bootstrap.sh;
              };
            in
            {
              User = haqqdUserName;
              Type = "oneshot";
              ExecStart = ''
                ${haqqd-init}/bin/haqqd-bootstrap
              '';
              LimitNOFILE = "infinity";
            };

          environment = {
            NIX_LD = config.environment.variables.NIX_LD;
          };

          before = [ "haqqd.service" ];
          after = [
            "network-online.target"
            "nss-lookup.target"
          ];
        };

        haqqd =
          {
            serviceConfig =
              let
                format = pkgs.formats.toml { };
                tomlConfig = format.generate "config.toml" (lib.attrsets.recursiveUpdate defaultCfgConfigToml cfg.config);
                appConfig = format.generate "app.toml" (lib.attrsets.recursiveUpdate defaultCfgAppToml cfg.app);
                start = pkgs.writeShellApplication {
                  name = "haqqd-start";
                  text = ''
                    ln -snf ${tomlConfig} .haqqd/config/config.toml
                    ln -snf ${appConfig} .haqqd/config/app.toml
                    ${pkgs.cosmovisor}/bin/cosmovisor run start
                  '';
                };
              in
              {
                User = haqqdUserName;
                ExecStart = ''${start}/bin/haqqd-start'';
                WorkingDirectory = cfg.userHome;
                Restart = "on-failure";
                RestartSec = 30;
                LimitNOFILE = "infinity";
              };

            environment = {
              DAEMON_HOME = "${cfg.userHome}/.haqqd";
              DAEMON_NAME = "haqqd";
              DAEMON_ALLOW_DOWNLOAD_BINARIES = "true";
              UNSAFE_SKIP_BACKUP = "false";

              NIX_LD = config.environment.variables.NIX_LD;
            };

            wantedBy = [ "multi-user.target" ];
            requires = [ "haqqd-bootstrap.service" ];
          };
      };
  };
}
