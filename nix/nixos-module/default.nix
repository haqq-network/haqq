{ ... }:
{
  imports = [
    ./haqqd-service.nix
    ./grafana-agent.nix
    ./delete-old-backups.nix
  ];
}
