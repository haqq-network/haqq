# Join TestEdge Tutorial

## Pick a Testnet

You specify the network you want to join by setting the **genesis file** and **seeds**. If you need more information about past networks, check our [testnets repo](https://github.com/haqq-network/testnets).

| Testnet Chain ID | Name | Version | Status | Description
|--|--|--|--|--|
| haqq_53211-1 | Haqq TestEdge | v1.0.3 | Live | This test network contains features which we plan to release on Haqq Mainnet. |
| haqq_112357-1 | Haqq TestNow | v1.0.3 | WIP | This test network is functionally equivalent to the current Haqq Mainnet and it built for developers and exchanges who are integrating with Haqq. |


## Install `haqqd`

Follow the [installation](./../quickstart/installation.md) document to install the {{ $themeConfig.project.name }} binary `{{ $themeConfig.project.binary }}`.

```
❗️Warning❗️
Make sure you have the right version of haqqd installed.
```

### Save Chain ID

We recommend saving the testnet `chain-id` into your `{{ $themeConfig.project.binary }}`'s `client.toml`. This will make it so you do not have to manually pass in the `chain-id` flag for every CLI command.

::: tip

See the Official [Chain IDs](./../basics/chain_id.md) for reference.

:::

### Cofigure TestEdge chain-id

```bash
haqqd config chain-id haqq_53211-1
```

## Initialize Node

We need to initialize the node to create all the necessary validator and node configuration files:

```bash
haqqd init <your_custom_moniker> --chain-id haqq_53211-1
```

::: danger

Monikers can contain only ASCII characters. Using Unicode characters will render your node unreachable.

:::

By default, the `init` command creates your `~/.haqqd` (i.e `$HOME`) directory with subfolders `config/` and `data/`.
In the `config` directory, the most important files for configuration are `app.toml` and `config.toml`.

## Genesis & Seeds

### Copy the Genesis File

Download genesis

```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/genesis.json
```

Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

Then verify the correctness of the genesis configuration file:

```bash
haqqd validate-genesis
```

Check binary version:

```sh
haqqd -v
# haqqd version "1.0.3" 58215364d5be4c9ab2b17b2a80cf89f10f6de38a
```

<!--### Add Seed Nodes-->

### Add Persistent Peers

We can set the [`persistent_peers`](https://docs.tendermint.com/master/tendermint-core/using-tendermint.html#persistent-peer) field in `~/.haqqd/config/config.toml` to specify peers that your node will maintain persistent connections with. You can retrieve them from the list of
available peers on the [`testnets`](https://github.com/haqq-network/testnets) repo.

 You can get a random 10 entries from the `peers.txt` file in the `PEERS` variable by running the following command:

```bash
PEERS=`curl -sL https://raw.githubusercontent.com/haqq-network/testnets/main/TestEdge/peers.txt | sort -R | head -n 10 | awk '{print $1}' | paste -s -d, -`
```

Use `sed` to include them into the configuration. You can also add them manually:

```bash
sed -i.bak -e "s/^persistent_peers *=.*/persistent_peers = \"$PEERS\"/" ~/.haqqd/config/config.toml
```

## Get a snapshot

### Prerequisites

Install lz4 if needed

```
apt-get install lz4
```

Download the snapshot:
```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/haqq_167797.tar.lz4
```

Decompress the snapshot to your database location. You database location will be something to the effect of ~/.haqqd depending on your node implemention.

```sh
lz4 -c -d haqq_167797.tar.lz4 | tar -x -C $HOME/.haqqd/data
```

## Pruning

The snapshot is designed for node opeartors to run an efficient validator service on Haqq chain. To make the snapshot as small as possible while still viable as a validator, we use the following setting to save on the disk space. We suggest you make the same adjustment on your node too. Please notice that your node will have very limited functionality beyond signing blocks with the efficient disk space utilization. For example, your node will not be able to serve as a RPC endpoint (which is not suggested to run on a validator node anyway).

Since we periodically state-sync our snapshot nodes, you might notice that sometimes the size of our snapshot is surprisingly small.

app.toml

```
pruning = "custom"
pruning-keep-recent = "100"
pruning-keep-every = "0"
pruning-interval = "10"
```

config.toml

```
indexer = "null"
```

## Run a Testnet Validator

Claim your testnet {{ $themeConfig.project.testnet_denom }} on the [faucet](./faucet.md) using your validator account address and submit your validator account address:

::: tip
For more details on how to run your validator, follow [these](./run_validator.md) instructions.
:::

```bash
haqqd tx staking create-validator \
  --amount=1000000000000aISLM \
  --pubkey=$(haqqd tendermint show-validator) \
  --moniker=<your_moniker_name> \
  --chain-id=<chain_id> \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1000000" \
  --gas="auto" \
  --gas-prices="0.025aISLM" \
  --from=<key_name>
```

## Start testnet

The final step is to [start the nodes](./../quickstart/run_node.md#start-node). Once enough voting power (+2/3) from the genesis validators is up-and-running, the testnet will start producing blocks.

```bash
haqqd start
```

## Staking delegate

You can deligate more ISLM to your stake.

```
haqqd tx staking delegate <your_validator_address> <quantity_ISLM> --from <key_name> -y
```

## Upgrading Your Node

::: tip
These instructions are for full nodes that have ran on previous versions of and would like to upgrade to the latest testnet version.
:::

### Reset Data

:::warning
If the version **new_version** you are upgrading to is not breaking from the previous one, you **should not** reset the data. If this is the case you can skip to [Restart](#restart)
:::

First, remove the outdated files and reset the data.

```bash
rm $HOME/.haqqd/config/addrbook.json $HOME/.haqqd/config/genesis.json
haqqd tendermint unsafe-reset-all --home $HOME/.haqqd
```

Your node is now in a pristine state while keeping the original `priv_validator.json` and `config.toml`. If you had any sentry nodes or full nodes setup before,
your node will still try to connect to them, but may fail if they haven't also
been upgraded.

::: danger Warning
Make sure that every node has a unique `priv_validator.json`. Do not copy the `priv_validator.json` from an old node to multiple new nodes. Running two nodes with the same `priv_validator.json` will cause you to double sign.
:::

### Restart

To restart your node, just type:

```bash
haqqd start
```
# Automated Upgrades

We are highly recommend use Cosmovisor for node upgrading. Learn how to automate chain upgrades using Cosmovisor. [upgrade](./upgrade.md)
