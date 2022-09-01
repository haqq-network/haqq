# Join TestEdge Tutorial

## Pick a Testnet

You specify the network you want to join by setting the **genesis file** and **seeds**. If you need more information about past networks, check our [testnets repo](https://github.com/haqq-network/testnets).

| Testnet Chain ID | Name | Version | Status | Description
|--|--|--|--|--|
| haqq_53211-1 | Haqq TestEdge | v1.0.3 | Live | This test network contains features which we plan to release on Haqq Mainnet. |
| haqq_112357-1 | Haqq TestNow | v1.0.3 | WIP | This test network is functionally equivalent to the current Haqq Mainnet and it built for developers and exchanges who are integrating with Haqq. |

## Preresquisites
- `make` & `gcc` 
- `Go 1.18+` [Install Go](https://go.dev/doc/install)

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

## Initialize Node

We need to initialize the node to create all the necessary validator and node configuration files:

```bash
haqqd init <your_moniker_name> --chain-id=<chain_id>
```

::: danger

Monikers can contain only ASCII characters. Using Unicode characters will render your node unreachable.

:::

By default, the `init` command creates your `~/.haqqd` (i.e `$HOME`) directory with subfolders `config/` and `data/`.
In the `config` directory, the most important files for configuration are `app.toml` and `config.toml`.

## Create keys

We recommend using [Tendermint KMS](./../guides/kms/kms.md) that allows separating key management from Tendermint nodes.
It is recommended that the KMS service runs in a separate physical hosts.

```sh
haqqd keys add <your_moniker_name>
```

## Configure chain-id

```sh
haqqd config chain-id <chain_id>
```

## Genesis & Seeds

To quickly get [started](./join.md#quickstart), node operators can choose to sync via State Sync (preferred) or by downloading a snapshot.

## Copy the Genesis File

### Download genesis

```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/genesis.json
```

### Update genesis file

```sh
mv genesis.json $HOME/.haqqd/config/genesis.json
```

### Verify the correctness of the genesis configuration file:

```bash
haqqd validate-genesis
```

### Check binary version:

```sh
haqqd -v
# haqqd version "1.0.3" 58215364d5be4c9ab2b17b2a80cf89f10f6de38a
```

<!--### Add Seed Nodes-->

## Add Persistent Peers

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

### Install lz4 if needed

```
apt-get install lz4
```

### Download the snapshot:

```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/haqq_latest.tar.lz4
```

Decompress the snapshot to your database location. You database location will be something to the effect of ~/.haqqd depending on your node implemention.

```sh
lz4 -c -d haqq_latest.tar.lz4 | tar -x -C $HOME/.haqqd
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


## Adding some ISLM to your account

### You can see your account address via execution this command

```sh
haqqd keys list

# - name: node01
#  type: local
#  address: haqq1jpl3c626gk08j5tkaja9g474xyxlmrqw726yzp
#  pubkey: '{"@type":"/ethermint.crypto.v1.ethsecp256k1.PubKey","key":"Aub31LXeh0OQrvR53FVhC21vSsO0Ab2Af0Fc4Otaeexi"}'
#  mnemonic: ""
```

### See different account address formats

```sh
haqqd debug addr <your_account_address>

# Address bytes: [144 127 28 105 90 69 158 121 81 118 236 186 84 87 213 49 13 253 140 14] 
# Address (hex): 907F1C695A459E795176ECBA5457D5310DFD8C0E 
# Address (EIP-55): 0x907F1c695a459E795176EcBA5457d5310dfd8C0e
# Bech32 Acc: haqq1jpl3c626gk08j5tkaja9g474xyxlmrqw726yzp
# Bech32 Val: haqqvaloper1jpl3c626gk08j5tkaja9g474xyxlmrqwjgk2xq
```

After that you can transfer some ISLM to your validator address.
If you don't have ISLM you can recive it using our [faucet](./../../testnet/faucet.md)

Claim your testnet ISLM on the [faucet](./faucet.md) using your validator account address and submit your validator account address:


### The next step is delegation ISLM to your validator

::: tip

Make sure you have already started the node.

```bash
haqqd start
```

:::


### Check your account balance

```sh
haqqd query bank balances <your_account_address>
# balances:
# - amount: "1009999000069343008" // equal to ~0.01 ISLM
#  denom: aISLM
# pagination:
#  next_key: null
#  total: "0"
```

Your `haqqvalconspub` can be used to create a new validator by staking tokens. You can find your validator pubkey by running:

```sh
haqqd tendermint show-address
```

## Staking delegate

Deligate some ISLM to your validator and make sure that deligation ISLM amount is not more than you have in balance.

```sh
haqqd tx staking delegate <your_account_tendermint_address> <quantity_ISLM> --from <your_key> -y
```

## Run a Testnet Validator

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
  --from=<key_name> \
  --node https://rpc.tm.testedge.haqq.network:443
```

## Upgrading Your Node

::: tip

These instructions are for full nodes that have ran on previous versions of and would like to upgrade to the latest testnet version.

:::

### Reset Data

:::warning

If the version **{new version}** you are upgrading to is not breaking from the previous one, you **should not**  reset the data. If this is the case you can skip to [Restart](#restart)

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

## Unjail Validator
When a validator is "jailed" for downtime, you must submit an Unjail transaction from the operator account in order to be able to get block proposer rewards again (depends on the zone fee distribution).

```sh
haaqd tx slashing unjail \
  --from=<key_name> \
  --chain-id=<chain_id>
```

## Common problems

You can read about common validator problems [here](./../guides/validators/setup.md#common-problems)

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
