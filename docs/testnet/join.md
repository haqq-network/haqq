<!--
order: 4
-->

# Haqq Network TestEdge

## Overview

The current Haqq version of TestEdge is [`v1.2.0`](https://github.com/haqq-network/haqq/releases/tag/v1.2.0). 

To bootstrap a TestEdge node, it is possible to sync from v1.2.0 via snapshot or via State Sync.

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

[Release page](https://github.com/haqq-network/haqq/releases/tag/v1.2.0) 

or

build from source 

```sh
cd $HOME && \
git clone -b v1.2.0 https://github.com/haqq-network/haqq && \
cd haqq && \
make install
```

To quickly get started, node operators can choose to sync via State Sync (preferred) or by downloading a snapshot.

:::: tabs

::: tab StateSync

**Run by Tendermint State Sync**

Check binary version:

```sh
haqqd -v
# haqqd version 1.2.0 40935b70fb1da4ee28f1d91e8601060e533f6fd0
```

Write your moniker name

```sh
CUSTOM_MONIKER="example_moniker"
```

Cofigure TestEdge chain-id

```sh
haqqd config chain-id haqq_54211-3
```

Initialize node

```sh
haqqd init CUSTOM_MONIKER --chain-id haqq_54211-3
```

Generate keys (if you haven't already done it)

We recommend using [Tendermint KMS](./../guides/kms/kms.md) that allows separating key management from Tendermint nodes.
It is recommended that the KMS service runs in a separate physical hosts.

```sh
haqqd keys add <name>
```

**Prepare genesis file for TestEdge**

Download genesis

```sh
curl -OL https://github.com/haqq-network/validators-contest/raw/master/genesis.json
```

Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

**Configure State sync**

Download script for state sync auto configuration

```sh
curl -OL https://raw.githubusercontent.com/haqq-network/testnets/main/TestEdge-2/state_sync.sh
```

List of seeds already included in script

```sh
SEEDS="62bf004201a90ce00df6f69390378c3d90f6dd7e@seed2.testedge2.haqq.network:26656,23a1176c9911eac442d6d1bf15f92eeabb3981d5@seed1.testedge2.haqq.network:26656"
```

Grant rights to the script and execute it

```sh
chmod +x state_sync.sh && ./state_sync.sh
```

**Start Haqq node**

```sh
haqqd start --x-crisis-skip-assert-invariants
```

:::

::: tab Snapshot

**Run from snapshot**

Check binary version:

```sh
haqqd -v
# haqqd version 1.2.0 40935b70fb1da4ee28f1d91e8601060e533f6fd0
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
haqqd config chain-id haqq_54211-3
```

Initialize node

```sh
haqqd init CUSTOM_MONIKER --chain-id haqq_54211-3
```

Generate keys (if you haven't already done it)

We recommend using [Tendermint KMS](./../guides/kms/kms.md) that allows separating key management from Tendermint nodes.
It is recommended that the KMS service runs in a separate physical hosts.

```sh
haqqd keys add <name>
```

**Prepare genesis file for TestEdge**

Download genesis

```sh
curl -OL https://github.com/haqq-network/validators-contest/raw/master/genesis.json
```

Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

**Unzip snapshot to data**

```sh
lz4 -c -d haqq_latest.tar.lz4 | tar -x -C $HOME/.haqqd
```

**Setup seeds**

```sh
SEEDS="62bf004201a90ce00df6f69390378c3d90f6dd7e@seed2.testedge2.haqq.network:26656,23a1176c9911eac442d6d1bf15f92eeabb3981d5@seed1.testedge2.haqq.network:26656"
```

```sh
sed -i.bak -E "s|^(seeds[[:space:]]+=[[:space:]]+).*$|\1\"$SEEDS\"|" $HOME/.haqqd/config/config.toml
```

**Start Haqq node**

```sh
haqqd start --x-crisis-skip-assert-invariants
```

:::

::: tab Sync from scratch

**Run with sync from scratch**

The main problem of synchronization from scratch is that we need to consistently change the version of the binary.
 
**Download binary v1.2.0 for your arch:**

```sh
https://github.com/haqq-network/haqq/releases/tag/v1.2.0 
```

or 

build from source

```sh
cd $HOME && \
git clone -b v1.2.0 https://github.com/haqq-network/haqq && \
cd haqq && \
make install
```

Check binary version:

```sh
haqqd -v
# haqqd version 1.2.0 40935b70fb1da4ee28f1d91e8601060e533f6fd0
```

**Start from v1.2.0:**

Write your moniker name

```sh
CUSTOM_MONIKER="example_moniker"
```

Cofigure TestEdge chain-id

```sh
haqqd config chain-id haqq_54211-3
```

Initialize node

```sh
haqqd init CUSTOM_MONIKER --chain-id haqq_54211-3
```

Generate keys (if you haven't already done it)

We recommend using [Tendermint KMS](./../guides/kms/kms.md) that allows separating key management from Tendermint nodes.
It is recommended that the KMS service runs in a separate physical hosts.

```sh
haqqd keys add <name>
```

**Prepare genesis file for TestEdge**

Download genesis

```sh
curl -OL https://github.com/haqq-network/validators-contest/raw/master/genesis.json
```

Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

**Setup seeds**

```sh
SEEDS="62bf004201a90ce00df6f69390378c3d90f6dd7e@seed2.testedge2.haqq.network:26656,23a1176c9911eac442d6d1bf15f92eeabb3981d5@seed1.testedge2.haqq.network:26656"
```

```sh
sed -i.bak -E "s|^(seeds[[:space:]]+=[[:space:]]+).*$|\1\"$SEEDS\"|" $HOME/.haqqd/config/config.toml
```

**Start Haqq**

```sh
haqqd start --x-crisis-skip-assert-invariants
```
Now wait until the chain reaches actual block height.

**Upgrade to Validator Node**

You now have an active full node. What's the next step? You can upgrade your full node to become a Haqq Validator. The top 150 validators have the ability to propose new blocks to the Haqq Network. Continue onto the [Validator Setup](https://docs.haqq.network/guides/validators/setup.html).

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
