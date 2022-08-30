<!--
order: 4
-->

# Validator Security Checklist

 Conduct a security checklist survey to go through the security measures of a validator {synopsis}

## Pre-requisite Readings

- [Validator Security](./security.md) {prereq}

## Conduct Survey on General Controls of Hosting Data Centre

Perform a survey on the hosting data centre, and compare your result with the best practice suggested below

For example, your hosting data centre should have following features

| Controls Category | Description of Best Practice    |
|-------------------|---------------------------------|
| Data Center       | Redundant Power                 |
| Data Center       | Redundant Cooling               |
| Data Center       | Redundant Networking            |
| Data Center       | Physical Cage/Gated Access      |
| Data Center       | Remote Alerting Security Camera |

## Current Status of Node Setup

Perform a survey on your current status of node setup, and compare your result with the best practice suggested below

| Controls Category                | Description of Best Practice                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
|----------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| General System Security          | Operating system appropriately patched. Kernel is updated to latest stable version. The node should be operated in x86_64 environment                                                                                                                                                                                                                                                                                                                                                                                                   |
| General System Security          | Auto-updates for operation system is configured. Toolkit for automatic upgrades exists (e.g. auter, yum-cron, dnf-automatic, unattended-upgrades)                                                                                                                                                                                                                                                                                                                                                                                       |
| General System Security          | Security framework enabled and enforcing. SELinux / AppArmor / Tomoyo / Grsecurity Enabled.                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| General System Security          | No insecure and unnecessary services Installed. (e.g. telnet, rsh, inetd, etc ...)                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| General System Security          | GRUB boot loader password is configured. Grub2 configured with password                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| General System Security          | Only root permissions on core system files                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| File Directory Security          | Secure the directory `~/.haqqd` to be accessible by owner only                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| Binary Configuration             | Recommend the following settings in config.toml for both performance and security - For **sentry nodes**: `max_num_inbound_peers = 500, max_num_outbound_peers = 50, flush_throttle_timeout = "300ms"` - For **validator node**: `max_num_inbound_peers = 100, max_num_outbound_peers = 10, flush_throttle_timeout = "100ms"`                                                                                                                                                                                                           |
| Account Security & Remote Access | Following Password policies are enforced: No Blank Passwords; Weak Passwords Not Allowed                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| Account Security & Remote Access | Following SSH configurations are enabled: PermitRootLogin: `no`; PasswordAuthentication `no`; ChallengeResponseAuthentication `no`; UsePAM `yes`; AllowUsers `Necessary user only`; AllowGroups `Necessary group only`.                                                                                                                                                                                                                                                                                                                 |
| Networking                       | Network throughput test using speedtest. Recommend to have at least 5 Mbps upload, 5 Mbps download)                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| Networking                       | Host-based (e.g. iptables) or cloud-based (e.g. AWS Security Group) firewall is enabled to protect all the involved nodes. Remote management ports (e.g. SSH - TCP 22) should only be exposed to selected IP instead of the internet. No overly permissive rules (e.g. wide range of allowed ports 1-65535) should be set. For internal communication channels between nodes, they should be set with specific source and destination addresses. For internet reachable nodes, set TCP 26656 to be the only incoming port, if possible. |
| Networking                       | Intrusion Detection / Prevention System (e.g. Fail2Ban, Snort, OSSEC) is installed and enforcing                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| Networking                       | Setup sentry node architecture to protect validator node, and set firewall rules to restrict direct internet access to it.                                                                                                                                                                                                                                                                                                                                                                                                              |
| Networking                       | The Remote Procedure Call (RPC) provides sensitive operations and information that is not supposed to be exposed to the Internet. By default, RPC is on and allow connection from 127.0.0.1 only. **Please be extremely careful** if you need to allow RPC from other IP addresses.                                                                                                                                                                                                                                                         |
| Redundancy                       | Hot standby node is setup with the same configuration as main node                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Redundancy                       | System monitoring and alerting is setup to alert owners on anomalies                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| Key Management                   | Setup [Tendermint KMS](./../kms/kms.md) with HSM or equivalent online service, which should replace the static key file.                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| DDOS                             | Setup validator in accordance with sentry architecture. Kindly refer to the setup [instruction](https://docs.tendermint.com/master/nodes/validators.html#setting-up-a-validator) and [detailed description](https://forum.cosmos.network/t/sentry-node-architecture-overview/454).                                                                                                                                                                                                                                                      |

## Summary

There is a few points about node security basic configuration:

- [ssh access](checklist.md#rule-for-ssh-connection)
- [limit network connections](checklist.md#limit-network-connections)
- [key security](checklist.md#key-security)

### What is a Firewall?

Firewall is a network security system that filters and controls the traffic on a predetermined set of rules. This is an intermediary system between the device and the internet.

There are a lot of ways to do this, but the easiest way I find is with iptables-persistent package. You can download the package from Ubuntu’s default repositories:

```sh
sudo apt-get update
```

```sh
sudo apt-get install iptables-persistent
```

Once the installation is complete, you can save your configuration using the command:

```sh
sudo invoke-rc.d iptables-persistent save
```

### Rule for ssh connection

:::danger

This is just an **example** 

:::

::: tip

We want to keep our SSH port open from the 192.168.1.3 network we blocked in the above case. That is we only want to allow those packets coming from 192.168.1.3 and which wants to go to the port 22.

:::

Execute the below command:

```sh
sudo iptables -A INPUT -s 192.168.1.3 -p tcp --dport 22 -j ACCEPT
```

The above command says looks for the packets originating from the IP address 192.168.1.3, having a TCP protocol and who wants to deliver something at the port 22 of my computer. If you find those packets then Accept them.

Now check the `iptable` configuration using `-L` flag.

```sh
sudo iptables -L
```

Therefore, any packet coming from 192.168.1.3 is first checked if it is going to the port 22 if it isn’t then it
is run through the next rule in the chain. Else it is allowed to pass the firewall.

## Limit network connections

The key idea is to restrict all connections and allow only used ports for security reasons.

You can find used ports in your node config files and make your own firewall rules based on your node setup:

```sh
~/.haqqd/config/app.toml
~/.haqqd/config/config.toml
~/.haqqd/config/client.toml
```

### Additional information

You can find more information about firewall configuration here:

- [Ubuntu security firewall](https://ubuntu.com/server/docs/security-firewall)

- [Iptables](https://help.ubuntu.com/community/IptablesHowTo)

- [firewalld](https://opensource.com/article/18/9/linux-iptables-firewalld)

## Key security

The `test` backend is a password-less variation of the file backend. Keys are stored unencrypted on disk. This keyring is provided for testing purposes only. Use at your own risk!

:::danger

🚨 DANGER: Never create your mainnet validator keys using a test keying backend. Doing so might result in a loss of funds by making your funds remotely accessible via the `eth_sendTransaction` JSON-RPC endpoint.

[Security Advisory: Insecurely configured geth can make funds remotely accessible](https://blog.ethereum.org/2015/08/29/security-alert-insecurely-configured-geth-can-make-funds-remotely-accessible)

:::

We recommend using [Tendermint KMS](./../kms/kms.md) that allows separating key management from Tendermint nodes.
It is recommended that the KMS service runs in a separate physical hosts.

## Sentry Node

Validators are responsible for ensuring that the network can sustain denial of service attacks.

One recommended way to mitigate these risks is for validators to carefully structure their network topology in a so-called sentry node architecture.

More information about sentry node you can find [here](./security.md#sentry-nodes-ddos-protection)

