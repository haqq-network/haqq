{ pkgs, lib, config, ... }:
let cfg = config.services.haqqd-supervised; in
{
  systemd.timers = lib.mkIf (cfg.enable && cfg.deleteOldBackups > 0)
    {
      haqqd-delete-old-backups = {
        timerConfig = {
          OnCalendar = "hourly";
          # OnCalendar = "*:0/1";
          Persistent = true;
          Unit = "haqqd-delete-old-backups.service";
        };

        wantedBy = [ "timers.target" ];
      };
    };

  systemd.services.haqqd-delete-old-backups = lib.mkIf (cfg.enable && cfg.deleteOldBackups > 0)
    {
      serviceConfig =
        let
          deleteOldBackups = pkgs.writeShellApplication {
            name = "haqqd-delete-old-backups";
            # buildInputs = [ pkgs.coreutils ];
            text = ''
              set -x
              find ${config.users.users.haqqd.home}/.haqqd -type d -name "data-backup-*" -mtime +${builtins.toString cfg.deleteOldBackups} -exec rm -rf {} +
            '';
          };
        in
        {
          User = config.users.users.haqqd.name;
          Type = "oneshot";
          ExecStart = ''
            ${deleteOldBackups}/bin/haqqd-delete-old-backups
          '';
        };
    };
}
