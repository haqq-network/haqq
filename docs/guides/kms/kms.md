<!--
order: 1
-->

# Tendermint KMS

Set up a Key Management System for Haqq {synopsis}

[Tendermint KMS](https://github.com/iqlusioninc/tmkms) is a Key Management Service (KMS) that allows separating key management from Tendermint nodes. In addition it provides other advantages such as:

- Improved security and risk management policies
- Unified API and support for various HSM (hardware security modules)
- Double signing protection (software or hardware based)

It is recommended that the KMS service runs in a separate physical hosts.

## Prepare TMKMS Dependencies

Start by opening the node you intend to run TMKMS (not the node you validate on) and install the following dependencies:
<br>

**Rust**

```sh
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```
```sh
source $HOME/.cargo/env
```
<br>

**GCC**
::::: tabs

:::: tab Ubuntu

```sh
sudo apt update
```
```sh
sudo apt install git build-essential ufw curl jq snapd --yes
```

::::

:::: tab Mac

```sh
brew install gcc
```
::::

:::::


**Libusb**


::::: tabs

:::: tab Ubuntu
```sh
apt install libusb-1.0-0-dev
```
::::

:::: tab Mac
```sh
brew install libusb
```
::::

:::::

<br>

::: tip

If on x86_64 architecture:

:::

```sh
export RUSTFLAGS=-Ctarget-feature=+aes,+ssse3
```

## Setup TMKMS

In this example, we will be compiling from the github source code using the `--features=softsign` flag, however you may use `--features=yubihsm` if you want to use a yubikey (ledger support is not working properly at the moment, and this guide will not go into using yubihsm).

```sh
cd $HOME
git clone https://github.com/iqlusioninc/tmkms.git
cd $HOME/tmkms
cargo install tmkms --features=softsign
tmkms init config
tmkms softsign keygen ./config/secrets/secret_connection_key
```

Now we will transfer your validator private key from your validator to your VM running TMKMS. You can do this manually or though scp. I will use scp in this example (the validator has the IP of 123.456.32.123):

```sh
scp user@123.456.32.123:~/.haqqd/config/priv_validator_key.json ~/tmkms/config/secrets
```

Then, import the private validator key into tmkms:

```sh
tmkms softsign import $HOME/tmkms/config/secrets/priv_validator_key.json $HOME/tmkms/config/secrets/priv_validator_key
```

Please note at this point, you could delete the `priv_validator_key.json` from both your validator node and tmkms node and store it safely offline in case of an emergency. This newly created `priv_validator_key` will be what TMKMS will use to sign for your validator.

Now, modify the `tmkms.toml` file

```sh
nano $HOME/tmkms/config/tmkms.toml
```
In this example, my validator has the IP address of 123.456.32.123 and we will be using port 688 to feed the validator key to the validator. We will also be using chain_id `haqq_11235-1` for Haqq Mainnet, but if you are doing this on the testnet be sure to use `haqq_53211-1` instead:

## Chain Configuration

**tmkms.toml**

::::: tabs

:::: tab Haqq Mainnet

```sh
# Tendermint KMS configuration file

## Chain Configuration

### Cosmos Hub Network

[[chain]]
id = "haqq_11235-1"
key_format = { type = "cosmos-json", account_key_prefix = "haqqpub", consensus_key_prefix = "haqqvalconspub" }
state_file = "/root/tmkms/config/state/priv_validator_state.json"

## Signing Provider Configuration

### Software-based Signer Configuration

[[providers.softsign]]
chain_ids = ["haqq_11235-1"]
key_type = "consensus"
path = "/root/tmkms/config/secrets/priv_validator_key"

## Validator Configuration

[[validator]]
chain_id = "haqq_11235-1"
addr = "tcp://123.456.32.123:688" # your validator node ip and port
secret_key = "/root/tmkms/config/secrets/secret_connection_key"
protocol_version = "v0.34"
reconnect = true

```

::::

:::: tab Haqq TestEdge

```sh
# Tendermint KMS configuration file

## Chain Configuration

### Cosmos Hub Network

[[chain]]
id = "haqq_53211-1"
key_format = { type = "cosmos-json", account_key_prefix = "haqqpub", consensus_key_prefix = "haqqvalconspub" }
state_file = "/root/tmkms/config/state/priv_validator_state.json"

## Signing Provider Configuration

### Software-based Signer Configuration

[[providers.softsign]]
chain_ids = ["haqq_53211-1"]
key_type = "consensus"
path = "/root/tmkms/config/secrets/priv_validator_key"

## Validator Configuration

[[validator]]
chain_id = "haqq_53211-1"
addr = "tcp://123.456.32.123:688" # your validator node ip and port
secret_key = "/root/tmkms/config/secrets/secret_connection_key"
protocol_version = "v0.34"
reconnect = true

```

::::

:::::

Now, modify your validators `config.toml` to use the port you selected in the `tmkms.toml` file:

```sh
nano $HOME/.haqqd/config/config.toml
```

```toml
priv_validator_laddr = "tcp://0.0.0.0:688"
```

It is also recommended to comment out the `priv_validator_key_file` line and the `priv_validator_state_file` line:

```sh
# Path to the JSON file containing the private key to use as a validator in the consensus protocol
# priv_validator_key_file = "config/priv_validator_key.json"

# Path to the JSON file containing the last sign state of a validator
# priv_validator_state_file = "data/priv_validator_state.json"
```

Next, stop the validator. Move back to your VM running TMKMS and start it:

```sh
tmkms start -c $HOME/tmkms/config/tmkms.toml
```

You will see error logs like the following:

```log
2022-03-08T23:42:37.926816Z  INFO tmkms::commands::start: tmkms 0.12.2 starting up...
2022-03-08T23:42:37.926968Z  INFO tmkms::keyring: [keyring:softsign] added consensus Ed25519 key: haqqvalconspub1zcjduepq2qfkp3ahrhaafzuqglme9mares0eluj58xr6cy7c37cdmzq0eecqk0yehr
2022-03-08T23:42:37.927216Z  INFO tmkms::connection::tcp: KMS node ID: 948f8fee83f7715f99b8b8a53d746ef00e7b0d9e
2022-03-08T23:42:37.929454Z ERROR tmkms::client: [haqq_11235-1@tcp://123.456.32.123:688] I/O error: Connection refused (os error 111)
2022-03-08T23:42:38.929746Z  INFO tmkms::connection::tcp: KMS node ID: 948f8fee83f7715f99b8b8a53d746ef00e7b0d9e
2022-03-08T23:42:38.931428Z ERROR tmkms::client: [haqq_11235-1@tcp://123.456.32.123:688] I/O error: Connection refused (os error 111)
2022-03-08T23:42:39.931729Z  INFO tmkms::connection::tcp: KMS node ID: 948f8fee83f7715f99b8b8a53d746ef00e7b0d9e
2022-03-08T23:42:39.932417Z ERROR tmkms::client: [haqq_11235-1@tcp://123.456.32.123:688] I/O error: Connection refused (os error 111)
2022-03-08T23:42:40.932732Z  INFO tmkms::connection::tcp: KMS node ID: 948f8fee83f7715f99b8b8a53d746ef00e7b0d9e
2022-03-08T23:42:40.933425Z ERROR tmkms::client: [haqq_11235-1@tcp://123.456.32.123:688] I/O error: Connection refused (os error 111)
```

Now, start your chornic validator on the validator node:

```sh
haqqd start
```

Your TMKMS node will now show logs like the following:

```log
2022-03-08T23:46:06.208451Z  INFO tmkms::connection::tcp: KMS node ID: 948f8fee83f7715f99b8b8a53d746ef00e7b0d9e
2022-03-08T23:46:06.210568Z  INFO tmkms::session: [haqq_11235-1@tcp://164.92.136.160:688] connected to validator successfully
2022-03-08T23:46:06.210604Z  WARN tmkms::session: [haqq_11235-1@tcp://164.92.136.160:688]: unverified validator peer ID! (ba44dd36899602e255b04e3608e5ef0fe4bc5f5b)
2022-03-08T23:46:15.929787Z  INFO tmkms::session: [haqq_11235-1@tcp://164.92.136.160:688] signed PreCommit:<nil> at h/r/s 3399910/0/2 (0 ms)
2022-03-08T23:46:17.344579Z  INFO tmkms::session: [haqq_11235-1@tcp://164.92.136.160:688] signed PreCommit:<nil> at h/r/s 3399911/0/2 (0 ms)
2022-03-08T23:46:22.367627Z  INFO tmkms::session: [haqq_11235-1@tcp://164.92.136.160:688] signed PreCommit:<nil> at h/r/s 3399912/0/2 (0 ms)
2022-03-08T23:46:27.409777Z  INFO tmkms::session: [haqq_11235-1@tcp://164.92.136.160:688] signed PreCommit:<nil> at h/r/s 3399913/0/2 (0 ms)
2022-03-08T23:46:32.442300Z  INFO tmkms::session: [haqq_11235-1@tcp://164.92.136.160:688] signed PreCommit:<nil> at h/r/s 3399914/0/2 (0 ms)
2022-03-08T23:46:37.452162Z  INFO tmkms::session: [haqq_11235-1@tcp://164.92.136.160:688] signed PreCommit:<nil> at h/r/s 3399915/0/2 (0 ms)
```

You should now be signing blocks! If you cancel the TMKMS process, you will no longer sign blocks and will stop syncing. If you restart the TMKMS process, your validator node will continue to sync from where it left off.

## Final Notes

Please note that this is a bare minimum setup. More robust settings such as setting up a firewall to only allow your TMKMS node to get through the priv_validator_laddr port would make your validator even more secure.
