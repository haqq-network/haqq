# Joining a Testnet

<!-- **🚧 `In developing...` 🏗️** -->

<!--
order: 4
-->

This document outlines the steps to join an existing testnet {synopsis}

## Pick a Testnet

You specify the network you want to join by setting the **genesis file** and **seeds**.

| Testnet Chain ID | Name | Version | Status | Description
|--|--|--|--|--| 
| haqq_112357-1 | Haqq TestNow | v1.0.0 | WIP | This test network is functionally equivalent to the current Haqq Mainnet and it built for developers and exchanges who are integrating with Haqq. |
| haqq_53211-1 | Haqq TestEdge | v1.0.0 | Live | This test network contains features which we plan to release on Haqq Mainnet. |


## Install `haqqd`

Follow the [installation](./quickstart/installation.md) document to install the {{ $themeConfig.project.name }} binary `{{ $themeConfig.project.binary }}`.

```
❗️Warning❗️
Make sure you have the right version of `{{ $themeConfig.project.binary }}` installed.
```

### Save Chain ID

We recommend saving the testnet `chain-id` into your `{{ $themeConfig.project.binary }}`'s `client.toml`. This will make it so you do not have to manually pass in the `chain-id` flag for every CLI command.

::: tip
See the Official [Chain IDs](./../users/technical_concepts/chain_id.md#official-chain-ids) for reference.
:::

```bash
haqqd config chain-id haqq_112357-1
```

## Initialize Node

We need to initialize the node to create all the necessary validator and node configuration files:

```bash
haqqd init <your_custom_moniker> --chain-id haqq_112357-1
```

::: danger
Monikers can contain only ASCII characters. Using Unicode characters will render your node unreachable.
:::

By default, the `init` command creates your `~/.haqqd` (i.e `$HOME`) directory with subfolders `config/` and `data/`.
In the `config` directory, the most important files for configuration are `app.toml` and `config.toml`.

<!-- TO DO ## Genesis & Seeds -->

<!-- ### Copy the Genesis File -->


<!-- ### Add Seed Nodes -->



<!-- ### Add Persistent Peers -->

## Run a Testnet Validator

Claim your testnet {{ $themeConfig.project.testnet_denom }} on the [faucet](./../developers/faucet.md) using your validator account address and submit your validator account address:

::: tip
For more details on how to run your validator, follow [these](./setup/run_validator.md) instructions.
:::

```bash
haqqd tx staking create-validator \
  --amount=1000000000000aISLM \
  --pubkey=$(haqqd tendermint show-validator) \
  --moniker="HaqqWhale" \
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

The final step is to [start the nodes](./quickstart/run_node.md#start-node). Once enough voting power (+2/3) from the genesis validators is up-and-running, the testnet will start producing blocks.

```bash
haqqd start
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