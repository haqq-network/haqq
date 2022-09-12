# Security Best Practices

This article is about a few points of node basic and additional security configurations {synopsis}

## Create a non-root user with `sudo` privileges

It is a good idea to log in as a non-root user. Logins with root permissions are often overlooked as a security risk, but this is not the case! The command `rm` can wipe your entire server if run incorrectly by a root user. By logging in as a non-root user, you can avoid this issue.

:::danger

Do NOT routinely use the root account. Please use `su` instead.

:::

Connect via SSH to your node

```bash
ssh username@server.public.ip.address
# example
# ssh ubuntu@17.23.161.11
```

Create a new user called haqq_node, for example

```sh
sudo useradd -m -s /bin/bash haqq_node
```

Set the password for haqq_node user

```sh
sudo passwd haqq_node
```

Add haqq_node to the sudo group

```sh
sudo usermod -aG sudo haqq_node
```

## Use SSH Keys only connection

The basic rules of hardening SSH are:

* Don't use password for SSH access (use private key)
* Don't allow root to SSH (the appropriate users should SSH in, then `su` or `sudo`)
* Use `sudo` for users so commands are logged
* Log unauthorized login attempts (and consider software to block/ban users who try to access your server too many times, for example fail2ban)
* Lock down SSH to only the ip range your require (optional)

You will need to create a key pair on your local machine, with the public key stored in the `keyname` file.

```sh
ssh-keygen -t ed25519
```

Transfer the public key `keyname.pub` to your remote node.

```bash
ssh-copy-id -i $HOME/.ssh/keyname.pub haqq_node@server.public.ip.address
```

Login with your new haqq_node user

```sh
ssh haqq_node@server.public.ip.address
```

Disable root login and password based login. Edit the `/etc/ssh/sshd_config file` using any text editor, for example `nano`

```sh
sudo nano /etc/ssh/sshd_config
```

Locate attributtes in `sshd_config`:

- `ChallengeResponseAuthentication` 
- `PasswordAuthentication`
- `PermitEmptyPasswords`

And modify their with `no` parameter

```sh
ChallengeResponseAuthentication no

PasswordAuthentication no

PermitEmptyPasswords no
```

And also find `PermitRootLogin` attributte. Edit it with `prohibit-password` parameter

```sh
PermitRootLogin prohibit-password
```

::: tip

**Optional**: You can also customize SSH **Port** with your **custom** numeric value.

Please [check for possible conflicts](https://en.wikipedia.org/wiki/List\_of\_TCP\_and\_UDP\_port\_numbers) first

:::

```bash
Port <port number>
```

Then validate the syntax of your new SSH configuration using this command

```sh
sudo sshd -t
```

If no errors with the syntax validation, restart the SSH process

```
sudo systemctl restart sshd
```

Verify that the ssh login still works

Standard SSH Port is `22`

```
ssh haqq_node@server.public.ip.address
```

Alternatively, you might need to add the `-p <port#>` flag if you used a custom SSH port.

```bash
ssh haqq_node@server.public.ip.address -p <custom port number>
```

Connection command

```bash
ssh -i <path to your SSH_key_name.pub> haqq_node@server.public.ip.address
```

**Optional**: Make logging in easier by updating your local ssh config.

You can also simplify the ssh command needed to log in to your server, consider updating your local `$HOME/.ssh/config` file:

```bash
Host ubuntu
  User haqq_node
  HostName <server.public.ip.address>
  Port <custom port number>
```

This will allow you to log in with `ssh haqq_node` rather than needing to pass through all ssh parameters explicitly.

## Update your system

::: danger

It's critically important to keep your system up-to-date with the latest patches to prevent intruders from accessing your system.

:::

```bash
sudo apt-get update -y && \
sudo apt dist-upgrade -y && \
sudo apt-get autoremove && \
sudo apt-get autoclean
```

::: tip

Enable automatic updates so you don't have to manually install them.

:::

```
sudo apt-get install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

## Disable root account

Use sudo execute to run commands as low-level users without requiring their own privileges.

```bash
# To disable the root account, simply use the -l option.
sudo passwd -l root
```

```bash
# If for some valid reason you need to re-enable the account, simply use the -u option.
sudo passwd -u root
```

## Setup Two Factor Authentication for SSH (optional)

SSH can be a great tool for connecting to remote Linux systems. What if you could add another layer of security. With `2FA`, you can create a another security layer and it can be implemented in several ways, for example it works well with Google Authenticator.

```sh
sudo apt install libpam-google-authenticator -y
```

To make SSH use the Google Authenticator PAM module, just edit the `/etc/pam.d/sshd` file:

```sh
sudo nano /etc/pam.d/sshd
```

Add this line:

```sh
auth required pam_google_authenticator.so
```

Now you need to restart the `sshd` daemon using:

```
sudo systemctl restart sshd.service
```

Modify `/etc/ssh/sshd_config`

```
sudo nano /etc/ssh/sshd_config
```

Find

- `ChallengeResponseAuthentication`
- `UsePAM`

And update this attributtes with `yes`

```sh
ChallengeResponseAuthentication yes

UsePAM yes
```

Save the file and exit.

Execute `google-authenticator` command.

```sh
google-authenticator
```

It will ask you a series of questions, here is one of recommended configuration:

* Make tokens “time-base”": yes
* Update the `.google_authenticator` file: `yes`
* Disallow multiple uses: `yes`
* Increase the original generation time limit: `no`
* Enable rate-limiting: `yes`

If you see the QR code and don’t have your phone, it will lead you to a website that has your emergency scratch codes printed on a card. You can keep the card in a safe place, so you won't need to dig around for it during an emergency.

Now, open `Google Authenticator` on your phone and add your secret key to make two factor authentication work.

:::danger

If you are enabling 2FA on a remote machine that you access over SSH you need to follow **steps 2 and 3** of [this tutorial](https://www.digitalocean.com/community/tutorials/how-to-set-up-multi-factor-authentication-for-ssh-on-ubuntu-18-04) to make 2FA work.

:::

## Secure Shared Memory (optional)

One of the first things you should do is secure the shared [memory](https://www.lifewire.com/what-is-random-access-memory-ram-2618159) used on the system. If you're unaware, shared memory can be used in an attack against a running service. Because of this, secure that portion of system memory.

To learn more about secure shared memory, read this [techrepublic.com article](https://www.techrepublic.com/article/how-to-enable-secure-shared-memory-on-ubuntu-server/).

## Install Fail2ban (optional)

Fail2ban is an intrusion prevention system that monitors log files and searches for particular patterns that correspond to failed login attempts. If a certain number of failed logins are detected from a specific IP address (within a specified amount of time), fail2ban blocks access from that IP address. Fail2ban works by automatically issuing the following actions against offending clients: iptables or ip6tables firewall rule, Fail2Ban action, efence iptables firewall rule, SSHGuard filter.

```sh
sudo apt-get install fail2ban -y
```

Then edit a config file that monitors SSH logins.

```sh
sudo nano /etc/fail2ban/jail.local
```

Add a following lines to the bottom of the file.

**Whitelisting IP address tip**: The `ignoreip` parameter accepts a list of IP addresses, IP ranges or DNS hosts that you can specify to be allowed to connect. This is where you want to specify your local machine, local IP range or local domain, separated by spaces.

```bash
# Example
ignoreip = 192.168.1.0/24 127.0.0.1/8 
```

```bash
[sshd]
enabled = true
port = <22 or your random port number>
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
# whitelisted IP addresses
ignoreip = <list of whitelisted IP address, your local daily laptop/pc>
```

Don't forget to save file. Restart fail2ban to take effect.

```
sudo systemctl restart fail2ban
```

## Configure your Firewall

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

### Simple firewall configuration for **validator**

After executing this command, we will create rules with allowed ports `22` and `26656`, the rest of the ports will be denied on `INPUT` chain:

* Port 22 (or your custom port) TCP for SSH connection
* Port 26656 tcp for p2p for incoming connections
* Deny other `INPUT` connections

```sh
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT && \
sudo iptables -A INPUT -p tcp --dport 26656 -j ACCEPT && \
sudo iptables -P INPUT DROP
```

Now you can check the `iptable` configuration using `-L` flag.

```sh
sudo iptables -L
```

::: tip

A more detailed description of used ports can be found below.

:::

### Rule for ssh connection

::: tip

We want to keep our SSH port open from the 192.168.1.3. That is we only want to allow those packets coming from 192.168.1.3 and which wants to go to the port 22.

:::

::: danger

If you don't have static IP address we don't recommend add to add this IP specific rule. For that case just allow SSH port (by default it use `22`)

```sh
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
```

:::


Execute the below command:

```sh
sudo iptables -A INPUT -s 192.168.1.3 -p tcp --dport 22 -j ACCEPT
```

The above command says looks for the packets originating from the IP address 192.168.1.3, having a TCP protocol and who wants to deliver something at the port 22 of your node. If you find those packets then Accept them.

Now check the `iptable` configuration using `-L` flag.

```sh
sudo iptables -L
```

Therefore, any packet coming from 192.168.1.3 is first checked if it is going to the port 22 if it isn’t then it
is run through the next rule in the chain. Else it is allowed to pass the firewall.

## Limit network connections

The key idea is to restrict all connections and allow only used ports for security reasons.

You can find used ports in your node config files and make your own firewall rules and based on your node setup:

```sh
~/.haqqd/config/app.toml
~/.haqqd/config/config.toml
~/.haqqd/config/client.toml
```

**Here is the best practice of used services:**

| port | service | rpc node | validator | sentry |
|---|---|---|---|---|
| 1317 | API server | ✅ | ❌ | ❌ |
| 8080 | Rosseta API | ✅ | ❌ | ❌ |
| 9090 | gRPC server | ✅ | ❌ | ❌ |
| 1317 | gRPC web | ✅ | ❌ | ❌ |
| 8545 | EMV RPC HTTP | ✅ | ❌ | ❌ |
| 8546 | EMV RPC WebSocket | ✅ | ❌ | ❌ |
| 26657 | Tendermint RPC | ✅ | ❌ | ❌ |
| 26658 | ABCI application | ❌ | ❌ | ❌ |
| 6060 | pprof (Go pkg) | ❌ | ❌ | ❌ |
| 26656 | p2p incoming connections | ✅ | ✅ | ✅ |
| 26660 | Prometheus | ❌ | ❌ | ❌ |

<br>

:::danger

For security reasons, you must block access to the listed ports in the table from the public network.

How to do it via [iptables](./security_best_practices.md#tcp-or-unix-socket-address-for-the-rpc-server-to-listen-on) or via [ufw](./security_best_practices.md#ufw-configuration-alternatively)

::: 

For example, you can check your config files for services used:

**File** `app.toml`

```yaml
###############################################################################
###                           gRPC Configuration                            ###
###############################################################################

[grpc]

# Enable defines if the gRPC server should be enabled.
enable = false
```


```yaml
###############################################################################
###                        gRPC Web Configuration                           ###
###############################################################################

[grpc-web]

# GRPCWebEnable defines if the gRPC-web should be enabled.
# NOTE: gRPC must also be enabled, otherwise, this configuration is a no-op.
enable = false
```

```yaml
###############################################################################
###                           JSON RPC Configuration                        ###
###############################################################################

[json-rpc]

# Enable defines if the gRPC server should be enabled.
enable = false
```

As you can see there are a few list of services, and we can have limit access to them via creating some specific rules:

Here is an examples of port specific rules:

### TCP or UNIX socket address for the RPC server to listen on

```sh
sudo iptables -A INPUT -p tcp --dport 26657 -j DROP
```

**TCP or UNIX socket address of the ABCI application,**
**or the name of an ABCI application compiled in with the Tendermint binary**

```sh
sudo iptables -A INPUT -p tcp --dport 26658 -j DROP
```

:::danger

We don't recommend limiting connections to this `port` (26656) because it is used in p2p communications between nodes.

```yaml
# Address to listen for incoming connections

laddr = "tcp://0.0.0.0:26656"
```

Removing all other traffic 

```sh
sudo iptables -A INPUT -j DROP
```

### Additional info

You can find additional information and examples [here](https://www.hostinger.com/tutorials/iptables-tutorial)

:::

### UFW configuration (alternatively)

The standard `UFW` firewall can be used to control network access to your node.

With any new installation, ufw is disabled by default. Enable it with the following settings.

* Port 22 (or your custom port) TCP for SSH connection
* Port 26656 tcp for p2p for incoming connections (necessary)

```bash
# By default, deny all incoming and outgoing traffic
sudo ufw default deny incoming
sudo ufw default allow outgoing
# Allow ssh access
sudo ufw allow ssh # port 22 or your custom ssh port number
# Allow necessary port
sudo ufw allow 26656/tcp

# Enable firewall
sudo ufw enable
```

```bash
# Verify status
sudo ufw status numbered
```

:::danger

Please don't exposed Grafana and Prometheus ports to the public internet. This introduces a new attack surface and would allow malicious attackers to access your data. You can built a secure solution with Wireguard, for example, and recommend you do so as well!

:::

If you are planning to use Grafana or (and) Prometheus don't forget to allow them too.

```bash
# Allow grafana web server port
sudo ufw allow 3000/tcp
# Enable prometheus endpoint port
sudo ufw allow 26660/tcp
```

**Optional but recommended** Whitelisting (or permitting connections from a specific IP) can be setup via the following command.

```bash
sudo ufw allow from <your local daily laptop/pc>
# Example
# sudo ufw allow from 192.168.50.22
```

:::tip

**Port Forwarding Tip:** You'll need to forward and open ports to your validator. Verify if it's working via using this [service](https://www.yougetsignal.com/tools/open-ports/) for example.

Or you can also use `netcat` (or nc in short) is a powerful and easy-to-use utility that can be employed for just about anything in Linux in relation to TCP, UDP, or UNIX-domain sockets.

:::

## Verify Listening Ports

If you want to maintain a secure server, you should validate the listening network ports every once in a while. This will provide you essential information about your network.

```bash
sudo ss -tulpn
```

Alternatively you can also use `netstat`

```bash
sudo netstat -tulpn
```

## VPN connection (optional)

:::danger

Recommended for Advanced Users Only

:::

It is good practice to arrange access to your node using a vpn connection or a secure tunnel and at the same time restrict access to the node from the external network.

You will learn how to set up secure and encrypt network traffic between two machines using WireGuard, greatly minimizing the chance of your local host being attacked by intruders and minimizing the attack surface of a remote host without requiring you to open ports for services like Grafana.

So, for example we can use WireGuard for secure connect with node.

### Install WireGuard

```sh
sudo apt install linux-headers-generic && \
sudo add-apt-repository ppa:wireguard/wireguard -y && \
sudo apt-get update && \
sudo apt-get install wireguard -y
```

### Setting Up Key Pairs

```sh
sudo su
cd /etc/wireguard
umask 077
wg genkey | tee wireguard-privatekey | wg pubkey > wireguard-publickey
```

Create a wg0.conf configuration file in /etc/wireguard directory.
Update your Private and Public Keys accordingly.
Change the Endpoint to your remote node public IP or DNS address.

Local machine config

```sh
# local node WireGuard Configuration
[Interface]
# local node address
Address = 10.0.0.1/32
# local node private key
PrivateKey = <i.e. SJ6ygM3csa36...+pO4XW1QU0B2M=>
# local node wireguard listening port
ListenPort = 51820

# remote node
[Peer]
# remote node's publickey
PublicKey = <i.e. Rq7QEe2g3qIjDftMu...knBGS9mvJDCa4WQg=>
# remote node's public ip address or dns address
Endpoint = remotenode.mydomainname.com:51820
# remote node's interface address
AllowedIPs = 10.0.0.2/32
PersistentKeepalive = 21
```

Node side config

```sh
# remote node WireGuard Configuration
[Interface]
Address = 10.0.0.2/32
PrivateKey = <i.e. cF3OjVhtKJAY/rQ...Fi7ASWg=>
ListenPort = 51820

# local node
[Peer]
# local node's public key
PublicKey = <i.e. rZLBzslvFtEJ...dfX4XSwk=>
# local node's public ip address or dns address
Endpoint = localnodesIP-or-domain.com:51820
# local node's interface address
AllowedIPs = 10.0.0.1/32
PersistentKeepalive = 21
```

### Configure Firewall

Configure `UFW` on both machines

```sh
sudo ufw allow 51820/udp
sudo ufw allow from 10.0.0.0/16 to any
# check the firewall rules
sudo ufw verbose
```

### Autostart

Add WireGuard service to systemd

```sh
sudo systemctl enable wg-quick@wg0.service
sudo systemctl daemon-reload
```

### Start service

Then you can start and check service

```sh
sudo systemctl start wg-quick@wg0
```

```sh
sudo systemctl status wg-quick@wg0
```

After that you can verify connect between machines

```sh
sudo wg

## Example Output
# interface: wg0
#  public key: rZLBzslvFtEJ...dfX4XSwk=
#  private key: (hidden)
#  listening port: 51820

#peer:
#  endpoint: 11.34.56.18:51820
#  allowed ips: 10.0.0.2/32
#  latest handshake: 11 seconds ago
#  transfer: 500 KiB received, 900 KiB sent
#  persistent keepalive: every 21 seconds
```

```sh
ping 10.0.0.2
```

To stop and disable WireGuard execute

```sh
sudo systemctl stop wg-quick@wg0 && \
sudo systemctl disable wg-quick@wg0.service && \
sudo systemctl daemon-reload
```

And of course you can also create your own firewall rules depending on the network and node configuration


## Additional best practices

|                        |                                                                                                                                                                                                                                                                             |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Networking             | In order to use ufw and Fail2ban's whitelisting feature, you need to assign static internal IPs to your validator node and daily laptop/PC. This is useful in conjunction with ufw and Fail2ban's whitelisting feature.        |
| Power Outage           | Make sure your nodes are backed up by turning on the Uninterruptible Power Supply (UPS), which will allow for power surges if there is a blackout. |
| Clear the bash history | <p>When pressing the up-arrow key, you can see prior commands which may contain sensitive data. To clear this, run the following:</p><p><code>shred -u ~/.bash_history && touch ~/.bash_history</code></p>                                                        |

## Additional information

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