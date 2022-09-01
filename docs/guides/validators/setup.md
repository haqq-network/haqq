<!--
order: 2
-->

# Run a Validator

Learn how to setup and run a validator node {synopsis}

## Pre-requisite Readings

::: tip

Pre-requisite Readings
1) [Validator Overview](./overview.md)
1) [Full Node Setup](../localnet/single_node.md#manual-localnet)

:::

If you plan to use a Key Management System (KMS), you should go through these steps first: [Using a KMS](./../kms/kms.md).

## What is a Validator?

[Validators](./overview.md) are responsible for committing new blocks to the blockchain through voting. A validator's stake is slashed if they become unavailable or sign blocks at the same height. Please read about [Sentry Node Architecture](https://hub.cosmos.network/main/validators/security.html) to protect your node from DDoS attacks and to ensure high-availability.

::: danger Warning
If you want to become a validator for the Hub's `mainnet`, you should [research security](./security.md).
:::

## Supported OS

We officially support Linux only. Other platforms may work but there is no
guarantee. We will extend our support to other platforms after we have stabilized our current
architecture.

## Minimum Requirements

To run testnet nodes, you will need a machine with the following minimum hardware requirements:

* 4 or more physical CPU cores
* At least 500GB of SSD disk storage
* At least 32GB of memory (RAM)
* At least 100mbps network bandwidth

As the usage of the blockchain grows, the server requirements may increase as well, so you should have a plan for updating your server as well.

## Create Your Validator

::: danger

**Before validator creation checklist 📋**

Before you started please make sure that you have been already done this steps for successful validator creation:

1) Init [node](./../../testnet/join_full.md#initialize-node)

1) Create [keys](./../../testnet/join_full.md#create-keys)

1) Configure [chain-id](./../../testnet/join_full.md#save-chain-id)

1) Added some ISLM to your [account](./../../testnet/join_full.md#adding-some-islm-to-your-account)

1) Deligated some ISLM to your [validator](./../../testnet/join_full.md#the-next-step-is-delegation-islm-to-your-validator)

1) Familiarize yourself with the best practices for node [security](./../validators/security_best_practices.md)

:::

## Additional information

You can found additional about joining TestEdge [here](./../../testnet/join_full.md)

## Create validator

To create your validator, just use the following command:

```bash
haqqd tx staking create-validator \
  --amount=1000000aISLM \
  --pubkey=$(haqqd tendermint show-validator) \
  --moniker="choose a moniker" \
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

::: tip

When specifying commission parameters, the `commission-max-change-rate` is used to measure % *point* change over the `commission-rate`. E.g. 1% to 2% is a 100% rate increase, but only 1 percentage point.

:::

::: tip
`Min-self-delegation` is a strictly positive integer that represents the minimum amount of self-delegated voting power your validator must always have. A `min-self-delegation` of `1000000` means your validator will never have a self-delegation lower than `1 aISLM`
:::

You can confirm that you are in the validator set by using a third party explorer.

<!-- ## Participate in Genesis as a Validator

If you want to participate in genesis as a validator, you need to justify that
you have some stake at genesis, create one (or multiple) transactions to bond this stake to your validator address, and include this transaction in the genesis file.

Your `haqqdvalconspub` can be used to create a new validator by staking tokens. You can find your validator pubkey by running:

```bash
haqqd tendermint show-validator
```

Next, craft your `haqqd gentx` command.

::: tip
A `gentx` is a JSON file carrying a self-delegation. All genesis transactions are collected by a `genesis coordinator` and validated against an initial `genesis.json`.
:::

```bash
haqqd gentx \
  --amount <amount_of_delegation_aISLM> \
  --commission-rate <commission_rate> \
  --commission-max-rate <commission_max_rate> \
  --commission-max-change-rate <commission_max_change_rate> \
  --pubkey <consensus_pubkey> \
  --name <key_name>
``` -->

::: tip

When specifying commission parameters, the `commission-max-change-rate` is used to measure % _point_ change over the `commission-rate`. E.g. 1% to 2% is a 100% rate increase, but only 1 percentage point.

:::

You can then submit your `gentx` on the [launch repository](https://github.com/cosmos/launch). These `gentx` will be used to form the final genesis file. 

## Edit Validator Description

You can edit your validator's public description. This info is to identify your validator, and will be relied on by delegators to decide which validators to stake to. Make sure to provide input for every flag below. If a flag is not included in the command the field will default to empty (`--moniker` defaults to the machine name) if the field has never been set or remain the same if it has been set in the past.

The <key_name> specifies which validator you are editing. If you choose to not include certain flags, remember that the --from flag must be included to identify the validator to update.

The `--identity` can be used as to verify identity with systems like Keybase or UPort. When using with Keybase `--identity` should be populated with a 16-digit string that is generated with a [keybase.io](https://keybase.io) account. It's a cryptographically secure method of verifying your identity across multiple online networks. The Keybase API allows us to retrieve your Keybase avatar. This is how you can add a logo to your validator profile.

```bash
haqqd tx staking edit-validator
  --moniker="choose a moniker" \
  --website="https://islamiccoin.net" \
  --identity=6A0D65E29A4CBC8E \
  --details="To infinity and beyond!" \
  --chain-id=<chain_id> \
  --gas="auto" \
  --gas-prices="0.025aISLM" \
  --from=<key_name> \
  --commission-rate="0.10"
```

__Note__: The `commission-rate` value must adhere to the following invariants:

- Must be between 0 and the validator's `commission-max-rate`
- Must not exceed the validator's `commission-max-change-rate` which is maximum
  % point change rate **per day**. In other words, a validator can only change
  its commission once per day and within `commission-max-change-rate` bounds.

## View Validator Description

View the validator's information with this command:

```bash
haqqd query staking validator <account_validator>
```

## Track Validator Signing Information

In order to keep track of a validator's signatures in the past you can do so by using the `signing-info` command:

```bash
haqqd query slashing signing-info <validator-pubkey>\
  --chain-id=<chain_id>
```

## Unjail Validator

When a validator is "jailed" for downtime, you must submit an `Unjail` transaction from the operator account in order to be able to get block proposer rewards again (depends on the zone fee distribution).

```bash
haqqd tx slashing unjail \
  --from=<key_name> \
  --chain-id=<chain_id>
```

## Confirm Your Validator is Running

Your validator is active if the following command returns anything:

```bash
haqqd query tendermint-validator-set | grep "$(haqqd tendermint show-address)"
```

You should now see your validator in one of Haqq explorers. You are looking for the `bech32` encoded `address` in the `~/.haqqd/config/priv_validator.json` file.

::: warning Note
To be in the validator set, you need to have more total voting power than the 100th validator.
:::

## Halting Your Validator

When attempting to perform routine maintenance or planning for an upcoming coordinated
upgrade, it can be useful to have your validator systematically and gracefully halt.
You can achieve this by either setting the `halt-height` to the height at which
you want your node to shutdown or by passing the `--halt-height` flag to `haqqd`.
The node will shutdown with a zero exit code at that given height after committing
the block.

## Common Problems

### Problem #1: My validator has `voting_power: 0`

Your validator has become jailed. Validators get jailed, i.e. get removed from the active validator set, if they do not vote on `500` of the last `10000` blocks, or if they double sign.

If you got jailed for downtime, you can get your voting power back to your validator. First, if `haqqd` is not running, start it up again:

```bash
haqqd start
```

Wait for your full node to catch up to the latest block. Then, you can [unjail your validator](#unjail-validator)

Lastly, check your validator again to see if your voting power is back.

```bash
haqqd status
```

You may notice that your voting power is less than it used to be. That's because you got slashed for downtime!

### Problem #2: My node crashes because of `too many open files`

The default number of files Linux can open (per-process) is `1024`. `haqqd` is known to open more than `1024` files. This causes the process to crash. A quick fix is to run `ulimit -n 4096` (increase the number of open files allowed) and then restart the process with `haqqd start`. If you are using `systemd` or another process manager to launch `haqqd` this may require some configuration at that level. A sample `systemd` file to fix this issue is below:

```toml
# /etc/systemd/system/haqqd.service
[Unit]
Description=Haqq Node
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu
ExecStart=/home/ubuntu/go/bin/haqqd start
Restart=on-failure
RestartSec=3
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
```

### Problem #3: My node crashes because of `validator set is nil in genesis and still empty after InitChain`

Make sure you have a genesis file in `$HOME/.haqqd/config/genesis.json` if you don't have this file you can find it here

```sh
curl -OL https://storage.googleapis.com/haqq-testedge-snapshots/genesis.json
```

And also you can validate genesis file using this command

```sh
haqqd validate-genesis
```

### Problem #4: I have an error while running validator `wrong Block.Header.AppHash.`

First of all, you should make sure that your bin file is up to date

```sh
haqqd -v
# haqqd version "1.0.3" 58215364d5be4c9ab2b17b2a80cf89f10f6de38a
```

We are currently using version `1.0.3` on TestEdge.

This error can also occur if you run the validator from a period when blocks were produced on a different version of the binary. 

From this point, we recommend starting the node using snapshot or statesync. More information you can find [here](./../../testnet/join_full.md)

### Unknown problems

If you encounter bugs that are not covered in our documentation portal, we'd love to see your bug report on our [discord](https://discord.gg/aZMm8pekhZ)

## Validator FAQ

If you have any problems with validator setting up you can visit our [Validator FAQ](./faq.md) page.