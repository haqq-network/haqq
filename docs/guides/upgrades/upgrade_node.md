<!--
order: 1
-->

# Upgrade Node

Learn how to upgrade your full node to the latest software version {synopsis}

With every new software release, we strongly recommend validators to perform a software upgrade, in order to prevent [double signing or halting the chain during consensus](https://docs.tendermint.com/master/spec/consensus/signing.html#double-signing).

You can upgrade your node by 1) upgrading your software version and 2) upgrading your node to that version. In this guide, you can find out how to automatically upgrade your node with Cosmovisor or perform the update manually.

## Software Upgrade

These instructions are for full nodes that have ran on previous versions of and would like to upgrade to the latest testnet.

First, stop your instance of `haqqd`. Next, upgrade the software:

```bash
cd haqq
git fetch --all && git checkout <new_version>
make install
```

::: tip
If you have issues at this step, please check that you have the latest stable version of GO installed.
:::

You will need to ensure that the version installed matches the one needed for th testnet. Check the Haqq [release page](https://github.com/haqq-network/haqq/releases) for details on each release.

Verify that everything is OK. If you get something like the following, you've successfully installed Haqq on your system.

```bash
$ haqqd version --long

name: haqq
server_name: haqqd
version: 0.4.0
commit: 070b668f2cbbf52548c46e96b236e09884483dd4
build_tags: netgo,ledger
go: go version go1.18 darwin/amd64
...
```

If the software version does not match, then please check your $PATH to ensure the correct haqqd is running.

## Upgrade Node

We highly recommend validators use Cosmovisor to run their nodes. This will make low-downtime upgrades smoother, as validators don't have to manually upgrade binaries during the upgrade. Instead users can preinstall new binaries, and cosmovisor will automatically update them based on on-chain Software Upgrade proposals.

You should review the docs for Cosmovisor located [here](https://docs.cosmos.network/master/run-node/cosmovisor.html)

If you choose to use Cosmovisor, please continue with these instructions. If you choose to upgrade your node manually instead, skip to the [the instructions without Cosmovisor](#upgrade-manually)

### Upgrade with Cosmovisor

> `cosmovisor` is a small process manager for Cosmos SDK application binaries that monitors the governance module for incoming chain upgrade proposals. If it sees a proposal that gets approved, cosmovisor can automatically download the new binary, stop the current binary, switch from the old binary to the new one, and finally restart the node with the new binary.

#### Install and Setup

To get started with [Cosmovisor](https://github.com/cosmos/cosmos-sdk/tree/master/cosmovisor) first download it

```bash
go get github.com/cosmos/cosmos-sdk/cosmovisor/cmd/cosmovisor
```

Set up the Cosmovisor environment variables. We recommend setting these in your `.profile` so it is automatically set in every session.

```bash
echo "# Setup Cosmovisor" >> ~/.profile
echo "export DAEMON_NAME=haqqd" >> ~/.profile
echo "export DAEMON_HOME=$HOME/.haqqd" >> ~/.profile
echo 'export PATH="$DAEMON_HOME/cosmovisor/current/bin:$PATH"' >> ~/.profile
source ~/.profile
```


After this, you must make the necessary folders for cosmosvisor in your daemon home directory (~/.haqqd).

```bash
mkdir -p ~/.haqqd/cosmovisor/upgrades
mkdir -p ~/.haqqd/cosmovisor/genesis/bin
cp $(which haqqd) ~/.haqqd/cosmovisor/genesis/bin/

# Verify the setup
# It should return the same version as haqqd
cosmovisor version
```

#### Preparing an Upgrade

Cosmovisor will continually poll the `$DAEMON_HOME/data/upgrade-info.json` for new upgrade instructions. When an upgrade is ready, node operators can download the new binary and place it under `$DAEMON_HOME/cosmovisor/upgrades/<name>/bin` where `<name>` is the URI-encoded name of the upgrade as specified in the upgrade module plan.

It is possible to have Cosmovisor automatically download the new binary. To do this set the following environment variable.

```bash
export DAEMON_ALLOW_DOWNLOAD_BINARIES=true
```

#### Download Genesis File

You can now download the "genesis" file for the chain. It is pre-filled with the entire genesis state and gentxs.

```bash
$ curl https://raw.githubusercontent.com/tharsis/testnets/main/olympus_mons/genesis.json > ~/.haqqd/config/genesis.json
```

We recommend using `sha256sum` to check the hash of the genesis.

```bash
cd ~/.haqqd/config
echo "2b5164f4bab00263cb424c3d0aa5c47a707184c6ff288322acc4c7e0c5f6f36f  genesis.json" | sha256sum -c
```

#### Reset Chain Database

There shouldn't be any chain database yet, but in case there is for some reason, you should reset it. This is a good idea especially if you ran `haqqd start` on an old, broken genesis file.

```bash
haqqd unsafe-reset-all
```

#### Ensure that you have set peers

In `~/.haqqd/config/config.toml` you can set your peers. See below for a list of up to date peers.

See the [Add persistent peers section](https://evmos.dev/testnet/join.html#add-persistent-peers) in our docs for an automated method, but field should look something like a comma separated string of peers (do not copy this, just an example):

```bash
persistent_peers = "b3ce1618585a9012c42e9a78bf4a5c1b4bad1123@65.21.170.3:33656,952b9d918037bc8f6d52756c111d0a30a456b3fe@213.239.217.52:29656,85301989752fe0ca934854aecc6379c1ccddf937@65.109.49.111:26556,d648d598c34e0e58ec759aa399fe4534021e8401@109.205.180.81:29956,f2c77f2169b753f93078de2b6b86bfa1ec4a6282@141.95.124.150:20116,eaa6d38517bbc32bdc487e894b6be9477fb9298f@78.107.234.44:45656,37513faac5f48bd043a1be122096c1ea1c973854@65.108.52.192:36656,d2764c55607aa9e8d4cee6e763d3d14e73b83168@66.94.119.47:26656,fc4311f0109d5aed5fcb8656fb6eab29c15d1cf6@65.109.53.53:26656,297bf784ea674e05d36af48e3a951de966f9aa40@65.109.34.133:36656,bc8c24e9d231faf55d4c6c8992a8b187cdd5c214@65.109.17.86:32656"
```

You can share your peer with

```bash
haqqd tendermint show-node-id
```

**Peer Format**: `node-id@ip:port`

**Example**: `3d892cfa787c164aca6723e689176207c1a42025@143.198.224.124:26656`

If you are relying on just seed node and no persistent peers or a low amount of them, please increase the following params in `config.toml`:

```bash
# Maximum number of inbound peers
max_num_inbound_peers = 200

# Maximum number of outbound peers to connect to, excluding persistent peers
max_num_outbound_peers = 100
```

#### Start your node

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

### Upgrade Manually

#### Upgrade Genesis File

:::warning
If the new version you are upgrading to has breaking changes, you will have to [export](#export-state) the state  and [restart](#restart-node) your node.

If it is **not** breaking (eg. from `v0.1.x` to `v0.1.<x+1>`), you can skip to [Restart](#restart-node) after installing the new version.
:::

To upgrade the genesis file, you can either fetch it from a trusted source or export it locally using the `haqqd export` command.

#### Fetch from a Trusted Source

If you are joining an existing testnet, you can fetch the genesis from the appropriate testnet source/repository where the genesis file is hosted.

Save the new genesis as `new_genesis.json`. Then, replace the old `genesis.json` with `new_genesis.json`.

```bash
cd $HOME/.haqqd/config
cp -f genesis.json new_genesis.json
mv new_genesis.json genesis.json
```


#### Export State

Haqq can dump the entire application state to a JSON file. This, besides upgrades, can be
useful for manual analysis of the state at a given height.

Export state with:

```bash
haqqd export > new_genesis.json
```

You can also export state from a particular height (at the end of processing the block of that height):

```bash
haqqd export --height [height] > new_genesis.json
```

If you plan to start a new network for 0 height (i.e genesis) from the exported state, export with the `--for-zero-height` flag:

```bash
haqqd export --height [height] --for-zero-height > new_genesis.json
```

Then, replace the old `genesis.json` with `new_genesis.json`.

```bash
cp -f genesis.json new_genesis.json
mv new_genesis.json genesis.json
```

At this point, you might want to run a script to update the exported genesis into a genesis state that is compatible with your new version.

You can use the `migrate` command to migrate from a given version to the next one (eg: `v0.X.X` to `v1.X.X`):

```bash
haqqd migrate [target-version] [/path/to/genesis.json] --chain-id=<new_chain_id> --genesis-time=<yyyy-mm-ddThh:mm:ssZ>
```

#### Restart Node

To restart your node once the new genesis has been updated, use the `start` command:

```bash
haqqd start
```
