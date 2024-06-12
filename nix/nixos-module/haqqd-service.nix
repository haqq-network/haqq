{
  pkgs,
  lib,
  config,
  ...
}:
with lib;
let
  cfg = config.services.haqqd;

  toml = pkgs.formats.toml { };

  importConfigFile = name: importTOML "${cfg.package}/share/haqqd/config/${name}.toml";
  defaultConfigTOML = importConfigFile "config";
  defaultAppTOML = importConfigFile "app";
  defaultClientTOML = importConfigFile "client";
in
{
  options.services.haqqd = {
    enable = mkEnableOption "";

    package = mkPackageOption pkgs "haqq" { };

    settings = {
      app = mkOption {
        inherit (toml) type;
        default = { };
      };

      config = mkOption {
        inherit (toml) type;
        default = { };
      };

      client = mkOption {
        inherit (toml) type;
        default = { };
      };
    };

    user = mkOption {
      type = types.str;
      default = "haqqd";
      description = "User account under which haqqd runs";
    };

    group = mkOption {
      type = types.str;
      default = "haqqd";
      description = "User group under which haqqd runs";
    };

    userHome = mkOption {
      type = types.str;
      default = "/var/lib/haqqd";
    };
  };

  config = mkIf cfg.enable {
    # FIXME: This is not ideal. We don't want to force the user to enable nix-ld
    # globally. We should run dynamically linked libraries and cosmovisir with
    # buildFHSEnv or some other kind of sandboxing/FHS-compatability layer.
    programs.nix-ld.enable = true;

    users = {
      users.${cfg.user} = {
        isSystemUser = true;
        home = cfg.userHome;
        createHome = true;
        inherit (cfg) group;
      };

      groups.${cfg.group} = { };
    };

    systemd.services = {
      haqqd = {
        path = with pkgs; [
          coreutils
          cosmovisor
          curl
          gnutar
          gzip
          haqq
        ];
        preStart =
          let
            generate =
              name: default: toml.generate "${name}.toml" (recursiveUpdate default cfg.settings.${name});
            appTOML = generate "app" defaultAppTOML;
            configTOML = generate "config" defaultConfigTOML;
            clientTOML = generate "client" defaultClientTOML;
          in
          ''
            if [ ! -f "$DAEMON_HOME/.bootstrapped" ]; then
              haqqd config chain-id "haqq_11235-1"
              haqqd init "haqq-node" --chain-id "haqq_11235-1"

              # Download mainnet genesis manifest.
              curl \
                -s \
                -L "https://raw.githubusercontent.com/haqq-network/mainnet/master/genesis.json" \
                -o "$DAEMON_HOME/config/genesis.json"

              # Download the genesis binary.
              curl \
                -s \
                -L "https://github.com/haqq-network/haqq/releases/download/v1.0.2/haqq_1.0.2_Linux_x86_64.tar.gz" \
                | tar xvz - -C /tmp \
                && install -Dm755 -t "$DAEMON_HOME/cosmovisor/genesis/bin" /tmp/bin/haqqd

              touch "$DAEMON_HOME/.bootstrapped"
            fi

            cp -f ${configTOML} "$DAEMON_HOME/config/config.toml"
            cp -f ${appTOML} "$DAEMON_HOME/config/app.toml"
            cp -f ${clientTOML} "$DAEMON_HOME/config/client.toml"
          '';
        script = ''
          cosmovisor run start
        '';
        environment = {
          DAEMON_HOME = "${cfg.userHome}/.haqqd";
          DAEMON_NAME = "haqqd";
          DAEMON_ALLOW_DOWNLOAD_BINARIES = "true";
          DAEMON_RESTART_AFTER_UPGRADE = "true";
          UNSAFE_SKIP_BACKUP = "false";
          inherit (config.environment.variables) NIX_LD;
        };
        serviceConfig = {
          User = cfg.user;
          Group = cfg.group;
          WorkingDirectory = cfg.userHome;
          Restart = "always";
          RestartSec = 5;
          LimitNOFILE = "infinity";
        };
      };
    };
  };
}
