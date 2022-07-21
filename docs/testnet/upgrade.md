# Automated Upgrades

Learn how to automate chain upgrades using Cosmovisor. {synopsis}

## Pre-requisites

- [Install Cosmovisor](https://docs.cosmos.network/main/run-node/cosmovisor.html#installation) {prereq}

## Using Cosmovisor

> `cosmovisor` is a small process manager for Cosmos SDK application binaries that monitors the governance module for incoming chain upgrade proposals. If it sees a proposal that gets approved, cosmovisor can automatically download the new binary, stop the current binary, switch from the old binary to the new one, and finally restart the node with the new binary.

::: tip
👉 For more info about Cosmovisor, please refer to the project official documentation [here](https://docs.cosmos.network/main/run-node/cosmovisor.html).
:::

We highly recommend validators use Cosmovisor to run their nodes. This will make low-downtime upgrades smoother, as validators don't have to [manually upgrade](./manual.md) binaries during the upgrade. Instead users can [pre-install](#manual-download) new binaries, and Cosmovisor will automatically update them based on on-chain Software Upgrade proposals.

### 1. Setup Cosmovisor

Set up the Cosmovisor environment variables. We recommend setting these in your `.profile` so it is automatically set in every session.

```bash
echo "# Setup Cosmovisor" >> ~/.profile
echo "export DAEMON_NAME=haqqd" >> ~/.profile
echo "export DAEMON_HOME=$HOME/.haqqd" >> ~/.profile
source ~/.profile
```

After this, you must make the necessary folders for `cosmosvisor` in your `DAEMON_HOME` directory (`~/.haqqd`) and copy over the current binary.

```bash
mkdir -p ~/.haqqd/cosmovisor
mkdir -p ~/.haqqd/cosmovisor/genesis
mkdir -p ~/.haqqd/cosmovisor/genesis/bin
mkdir -p ~/.haqqd/cosmovisor/upgrades

cp $GOPATH/bin/haqqd ~/.haqqd/cosmovisor/genesis/bin
```

To check that you did this correctly, ensure your versions of `cosmovisor` and `haqqd` are the same:

```bash
cosmovisor version
haqqd version
```

### 2. Download the Haqq release

#### 2.a) Manual Download

Cosmovisor will continually poll the `$DAEMON_HOME/data/upgrade-info.json` for new upgrade instructions. When an upgrade is [released](https://github.com/haqq-network/haqq/releases), node operators need to:

1. Download (**NOT INSTALL**) the binary for the new release
2. Place it under `$DAEMON_HOME/cosmovisor/upgrades/<name>/bin`, where `<name>` is the URI-encoded name of the upgrade as specified in the Software Upgrade Plan.

```

Your `cosmovisor/` directory should look like this:

```shell
cosmovisor/
├── current/   # either genesis or upgrades/<name>
├── genesis
│   └── bin
│       └── haqqd
└── upgrades
    └── v1.0.3
        ├── bin
        │   └── haqqd
        └── upgrade-info.json
```

#### 2.b) Automatic Download

::: warning
**NOTE**: Auto-download doesn't verify in advance if a binary is available. If there will be any issue with downloading a binary, `cosmovisor` will stop and won't restart an the chain (which could lead it to a halt).
:::

It is possible to have Cosmovisor [automatically download](https://docs.cosmos.network/main/run-node/cosmovisor.html#auto-download) the new binary. Validators can use the automatic download option to prevent unnecessary downtime during the upgrade process. This option will automatically restart the chain with the upgrade binary once the chain has halted at the proposed `upgrade-height`. The major benefit of this option is that validators can prepare the upgrade binary in advance and then relax at the time of the upgrade.

To set the auto-download use set the following environment variable:

```bash
echo "export DAEMON_ALLOW_DOWNLOAD_BINARIES=true" >> ~/.profile
```

### 3. Start your node

Now that everything is setup and ready to go, you can start your node.

```bash
cosmovisor start
```

You will need some way to keep the process always running. If you're on linux, you can do this by creating a service.

```bash
sudo tee /etc/systemd/system/haqqd.service > /dev/null <<EOF
[Unit]
Description=Haqq Daemon
After=network-online.target

[Service]
User=$USER
ExecStart=$(which cosmovisor) start
Restart=always
RestartSec=3
LimitNOFILE=infinity

Environment="DAEMON_HOME=$HOME/.haqqd"
Environment="DAEMON_NAME=haqqd"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"

[Install]
WantedBy=multi-user.target
EOF
```

Then update and start the node

```bash
sudo -S systemctl daemon-reload
sudo -S systemctl enable haqqd
sudo -S systemctl start haqqd
```

You can check the status with:

```bash
systemctl status haqqd
```

# Manual Upgrades

Learn how to manually upgrade your node. {synopsis}

## Pre-requisites

- [Install Haqq](./../quickstart/installation.md) {prereq}

## 1. Upgrade the Haqq version

Before upgrading the Haqq version. Stop your instance of `haqqd` using `Ctrl/Cmd+C`.

Next, upgrade the software to the desired release version. Check the Haqq [releases page](https://github.com/haqq-network/haqq/releases) for details on each release.

::: warning
Ensure that the version installed matches the one needed for the network you are running (mainnet or testnet).
:::

```bash
cd haqq
git fetch --all && git checkout <new_version>
make install
```

::: tip
If you have issues at this step, please check that you have the latest stable version of [Golang](https://golang.org/dl/) installed.
:::

Verify that you've successfully installed Haqq on your system by using the `version` command:

```bash
$ haqqd version --long

name: haqqd
server_name: haqqd
version: 1.0.3
commit: fe9df43332800a74a163c014c69e62765d8206e3
build_tags: netgo,ledger
go: go version go1.18 darwin/amd64
...
```

::: tip
If the software version does not match, then please check your `$PATH` to ensure the correct `haqqd` is running.
:::

## 2. Replace Genesis file

::: tip
You can find the latest `genesis.json` file for mainnet or testnet in the following repositories:

- **Testnet**: [github.com/haqq-network/testnets](https://github.com/haqq-network/testnets)
:::

Save the new genesis as `new_genesis.json`. Then, replace the old `genesis.json` located in your `config/` directory with `new_genesis.json`:

```bash
cd $HOME/.haqqd/config
cp -f genesis.json new_genesis.json
mv new_genesis.json genesis.json
```

::: tip
We recommend using `sha256sum` to check the hash of the downloaded genesis against the expected genesis.

```bash
cd ~/.haqqd/config
echo "<expected_hash>  genesis.json" | sha256sum -c
```

:::

## 3. Data Reset

::: danger
Check [here](./upgrades.md) if the version you are upgrading require a data reset (hard fork). If this is not the case, you can skip to [Restart](#restart-node).
:::

Remove the outdated files and reset the data:

```bash
rm $HOME/.haqqd/config/addrbook.json
haqqd tendermint unsafe-reset-all --home $HOME/.haqqd
```

Your node is now in a pristine state while keeping the original `priv_validator.json` and `config.toml`. If you had any sentry nodes or full nodes setup before,
your node will still try to connect to them, but may fail if they haven't also
been upgraded.

::: danger
🚨 **IMPORTANT** 🚨

Make sure that every node has a unique `priv_validator.json`. **DO NOT** copy the `priv_validator.json` from an old node to multiple new nodes. Running two nodes with the same `priv_validator.json` will cause you to [double sign](https://docs.tendermint.com/master/spec/consensus/signing.html#double-signing).
:::

## 4. Restart Node

To restart your node once the new genesis has been updated, use the `start` command:

```bash
haqqd start
```