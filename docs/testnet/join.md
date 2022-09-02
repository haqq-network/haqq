<!--
order: 4
-->

# Haqq Network TestEdge

## Overview

The current Haqq version of TestEdge is [`v1.0.3`](https://github.com/haqq-network/haqq/releases/tag/v1.0.3). 

To bootstrap a TestEdge node, it is possible to sync from v1.0.3 via snapshot or via State Sync.

This document outlines the steps to join an existing testnet {synopsis}

## Quickstart

Install packages:

```sh
sudo apt-get install curl git make gcc liblz4-tool build-essential jq -y
```

**Preresquisites for compile from source**
- `make` & `gcc` 
- `Go 1.18+` [Install Go](https://go.dev/doc/install)


::: tip

**If you choose StateSync or Snapshot for node running you need to download or build from source**

:::

Build from source:

Download latest binary for your arch:

[Release page](https://github.com/haqq-network/haqq/releases/tag/v1.0.3) 

or

build from source 

```sh
cd $HOME && \
git clone -b v1.0.3 https://github.com/haqq-network/haqq && \
cd haqq && \
make install
```

To quickly get started, node operators can choose to sync via State Sync (preferred) or by downloading a snapshot.

:::: tabs

::: tab StateSync

### Run by Tendermint State Sync

Check binary version:

```sh
haqqd -v
# haqqd version "1.0.3" 58215364d5be4c9ab2b17b2a80cf89f10f6de38a
```

Write your moniker name

```sh
CUSTOM_MONIKER="example_moniker"
```

Cofigure TestEdge chain-id

```sh
haqqd config chain-id haqq_53211-1
```

Initialize node

```sh
haqqd init CUSTOM_MONIKER --chain-id haqq_53211-1
```

Generate keys (if you haven't already done it)

We recommend using [Tendermint KMS](./../guides/kms/kms.md) that allows separating key management from Tendermint nodes.
It is recommended that the KMS service runs in a separate physical hosts.

```sh
haqqd keys add <name>
```

### Prepare genesis file for TestEdge

Download genesis

```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/genesis.json
```

Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

### Configure State sync

Download script for state sync auto configuration

```sh
curl -OL https://raw.githubusercontent.com/haqq-network/testnets/main/TestEdge/state_sync.sh
```

Grant rights to the script and execute it

```sh
chmod +x state_sync.sh && ./state_sync.sh
```

### Start Haqq node

```sh
haqqd start --x-crisis-skip-assert-invariants
```

:::

::: tab Snapshot

### Run from snapshot

Check binary version:

```sh
haqqd -v
# haqqd version "1.0.3" 58215364d5be4c9ab2b17b2a80cf89f10f6de38a
```

Download the snapshot:
```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/haqq_latest.tar.lz4
```

Write your moniker name

```sh
CUSTOM_MONIKER="example_moniker"
```

Cofigure TestEdge chain-id

```sh
haqqd config chain-id haqq_53211-1
```

Initialize node

```sh
haqqd init CUSTOM_MONIKER --chain-id haqq_53211-1
```

Generate keys (if you haven't already done it)

We recommend using [Tendermint KMS](./../guides/kms/kms.md) that allows separating key management from Tendermint nodes.
It is recommended that the KMS service runs in a separate physical hosts.

```sh
haqqd keys add <name>
```

### Prepare genesis file for TestEdge

Download genesis

```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/genesis.json
```

Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

### Unzip snapshot to data

```sh
lz4 -c -d haqq_latest.tar.lz4 | tar -x -C $HOME/.haqqd
```

### Setup seeds

```sh
SEEDS="ddc217640ab137ad6f9cf11fd94fba02eb1e1972@seed1.testedge.haqq.network:26656,7028d26e4d37506b4d5e1f668c945a93693d111b@seed2.testedge.haqq.network:26656"
```

```sh
sed -i.bak -E "s|^(seeds[[:space:]]+=[[:space:]]+).*$|\1\"$SEEDS\"|" $HOME/.haqqd/config/config.toml
```

### Start Haqq node

```sh
haqqd start --x-crisis-skip-assert-invariants
```

:::

::: tab Sync from scratch

### Run with sync from scratch

The main problem of synchronization from scratch is that we need to consistently change the version of the binary.
Currently we need upgrades binary by this pipepline:
v1.0.1 -> v1.0.2 -> v1.0.3

### Download binary v1.0.1 for your arch:

```sh
https://github.com/haqq-network/haqq/releases/tag/v1.0.1 
```

or 

build from source

```sh
cd $HOME && \
git clone -b v1.0.1 https://github.com/haqq-network/haqq && \
cd haqq && \
make install
```

Check binary version:

```sh
haqqd -v
# haqqd version "1.0.3" 58215364d5be4c9ab2b17b2a80cf89f10f6de38a
```

### Start from v1.0.1:

Write your moniker name

```sh
CUSTOM_MONIKER="example_moniker"
```

Cofigure TestEdge chain-id

```sh
haqqd config chain-id haqq_53211-1
```

Initialize node

```sh
haqqd init CUSTOM_MONIKER --chain-id haqq_53211-1
```

Generate keys (if you haven't already done it)

We recommend using [Tendermint KMS](./../guides/kms/kms.md) that allows separating key management from Tendermint nodes.
It is recommended that the KMS service runs in a separate physical hosts.

```sh
haqqd keys add <name>
```

### Prepare genesis file for TestEdge

Download genesis

```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/genesis.json
```

Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

### Setup seeds

```sh
SEEDS="ddc217640ab137ad6f9cf11fd94fba02eb1e1972@seed1.testedge.haqq.network:26656,7028d26e4d37506b4d5e1f668c945a93693d111b@seed2.testedge.haqq.network:26656"
```

```sh
sed -i.bak -E "s|^(seeds[[:space:]]+=[[:space:]]+).*$|\1\"$SEEDS\"|" $HOME/.haqqd/config/config.toml
```

### Start Haqq

```sh
haqqd start --x-crisis-skip-assert-invariants
```

Now wait until the chain reaches block height 1143. It will panic and log the following:

```
panic: UPGRADE "v1.0.2" NEEDED at height: 1143
```

It's now time to perform the manual upgrade to `v1.0.2`:

```sh
git checkout v1.0.2 && \
make install && \
haqqd start --x-crisis-skip-assert-invariants
```

Now wait until the chain reaches block height 1928. It will panic and log the following:
```
panic: UPGRADE "v1.0.3" NEEDED at height: 1928
```

It's now time to perform the manual upgrade to `v1.0.3`:

```sh
git checkout v1.0.3 && \
make install && \
haqqd start --x-crisis-skip-assert-invariants
```

### Upgrade to Validator Node

You now have an active full node. What's the next step? You can upgrade your full node to become a Haqq Validator. The top 100 validators have the ability to propose new blocks to the Haqq Network. Continue onto the [Validator Setup](https://docs.haqq.network/guides/validators/setup.html).

```sh
tar cvf - $HOME/haqq_backups/haqq_$LATEST_HEIGHT/ | lz4 - $HOME/haqq_backups/haqq_$LATEST_HEIGHT.tar.lz4
```

```sh
cd ~/haqq_backups/haqq_167797
```

:::

::::

For more advanced information on setting up a node, see the full [Join TestEdge Tutorial](./join_full.md)

## Validators FAQ

If you have any problems with validator setting up you can visit our [Validator FAQ](./../guides/validators/faq.md) page.

## Validator Security

:::danger

Before starting a node, we recommend that you read the following articles in order to ensure security and prevent gaining access to a node and your test coins.

[Validator Security](./../guides/validators/security.md)

[Validator Security Checklist](./../guides/validators/checklist.md)

[Security Best Practices](./../guides/validators/security_best_practices.md)

[Tendermint KMS](./../guides/kms/kms.md)

:::

## Automated Upgrades

We are highly recommend use Cosmovisor for node upgrading. Learn how to automate chain upgrades using [Cosmovisor](./upgrade.md)

### Validators community

We are always welcome you as one of our validators. You can join and interacting with other members in our [Discord](https://discord.gg/aZMm8pekhZ).