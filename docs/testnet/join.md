<!--
order: 4
-->

# Haqq Network TestEdge2

## Overview

The current Haqq version of TestEdge2 is [`v1.3.0`](https://github.com/haqq-network/haqq/releases/tag/v1.3.0). 

To bootstrap a TestEdge node, it is possible to sync from v1.3.0 via snapshot or via State Sync.

This document outlines the steps to join an existing testnet {synopsis}

## Quickstart

Install packages:

```sh
sudo apt-get install curl git make gcc liblz4-tool build-essential jq bzip2 -y
```

**Preresquisites for compile from source**
- `make` & `gcc` 
- `Go 1.19+` [Install Go](https://go.dev/doc/install)

Build from source:

Download latest binary for your arch:

[Release page](https://github.com/haqq-network/haqq/releases/tag/v1.3.0) 

or

build from source 

```sh
cd $HOME && \
git clone -b v1.3.0 https://github.com/haqq-network/haqq && \
cd haqq && \
make install
```

To quickly get started, node operators can choose to sync via State Sync (preferred) or by downloading a snapshot.

**Run by Tendermint State Sync**

Check binary version:

```sh
haqq@haqq-node:~# haqqd -v
haqqd version "1.3.0" a9a92e5e524ca651204f8ee6ebe5d1b12e4519b3
```

Write your moniker name

```sh
CUSTOM_MONIKER="haqq_node_testedge2"
```

Cofigure TestEdge2 chain-id

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

**Prepare genesis file for TestEdge2**

Download genesis zip file

```sh
curl -OL curl -OL https://raw.githubusercontent.com/haqq-network/testnets/main/TestEdge2/genesis.tar.bz2 
```

Unzip file

```sh
bzip2 -d genesis.tar.bz2 && tar -xvf genesis.tar 
```

Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

**Configure State sync**

Download script for state sync auto configuration

```sh
curl -OL https://raw.githubusercontent.com/haqq-network/testnets/main/TestEdge2/state_sync.sh
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
haqqd start
```

**Upgrade to Validator Node**

You now have an active full node. What's the next step? You can upgrade your full node to become a Haqq Validator. The top 100 validators have the ability to propose new blocks to the Haqq Network. Continue onto the [Validator Setup](https://docs.haqq.network/guides/validators/setup.html).


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
