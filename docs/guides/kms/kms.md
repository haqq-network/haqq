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

## Install Tendermint KMS onto the node

You will need the following prerequisites:

- ✅ **Rust** (stable; **1.56+**): https://rustup.rs/
- ✅ **C compiler**: e.g. gcc, clang
- ✅ **pkg-config**
- ✅ **libusb** (1.0+). Install instructions for common platforms
    - ✅ Debian/Ubuntu

      ```bash
      apt install libusb-1.0-0-dev
      ```

    - ✅ RedHat/CentOS
  
      ```bash
      yum install libusb1-devel
      ```

    - ✅ macOS (Homebrew)
  
      ```
      brew install libusb
      ```

::: tip
For `x86_64` architecture only:

Configure `RUSTFLAGS` environment variable:

```bash
export RUSTFLAGS=-Ctarget-feature=+aes,+ssse3
```

:::

We are ready to install KMS. There are 2 ways to do this: compile from source or install with Rusts cargo-install. We’ll use the [second](./kms.md#tmkms-setup) option.

### Compile from source code

The following example adds `--features=ledger` to enable Ledger  support.
`tmkms` can be compiled directly from the git repository source code, using the following commands:

```bash
gh repo clone iqlusioninc/tmkms && cd tmkms
[...]
cargo build --release --features=ledger
```

Alternatively, substitute `--features=yubihsm` to enable [YubiHSM](https://www.yubico.com/products/hardware-security-module/) support.

If successful, it will produce the `tmkms` executable located at: `./target/release/tmkms`.

### YubiHSM
  
Detailed information on how to setup a KMS with [YubiHSM 2](https://www.yubico.com/products/hardware-security-module/) can be found [here](https://github.com/iqlusioninc/tmkms/blob/master/README.yubihsm.md).

<!--### Ledger Tendermint app

Detailed information on how to setup a KMS with Ledger Tendermint App can be found [here](kms_ledger.md).

-->

## TMKMS setup

### Install TMKMS

```sh
sudo apt install build-essential && \
curl https://sh.rustup.rs -sSf | sh && \
source $HOME/.cargo/env && \
cargo install tmkms --features=softsign
```

### Init TMKMS config directory

```sh
sudo mkdir -p /etc/tmkms && \
sudo chown sa_100720845795915073218:sa_100720845795915073218 /etc/tmkms
```

## Configure TMKMS

### Init TMKMS

```sh
tmkms init /etc/tmkms/tmkms01
```

### Import keys

```sh
tmkms softsign import ~/val01/config/priv_validator_key.json /etc/tmkms/tmkms01/secrets/validator.key
```

### Create config template

**File ~/tmkms.tpl.yaml**

```yaml
# Tendermint KMS configuration file

## Chain Configuration

### Cosmos Hub Network

[[chain]]
id = "{{ CHAIN_ID }}"
key_format = { type = "cosmos-json", account_key_prefix = "haqqpub", consensus_key_prefix = "haqqvalconspub" }
state_file = "/etc/tmkms/tmkms{{ VAL_N }}/state/{{ CHAIN_ID }}-consensus.json"

## Signing Provider Configuration

### Software-based Signer Configuration

[[providers.softsign]]
chain_ids = ["{{ CHAIN_ID }}"]
key_type = "consensus"
path = "/etc/tmkms/tmkms{{ VAL_N }}/secrets/validator.key"

## Validator Configuration

[[validator]]
chain_id = "{{ CHAIN_ID }}"
addr = "tcp://{{ VAL_IP }}:29750"
secret_key = "/etc/tmkms/tmkms{{ VAL_N }}/secrets/kms-identity.key"
protocol_version = "v0.34"
reconnect = true
```

### Deploy configs

```sh
cat ~/tmkms.tpl.yaml | sed 's/{{ CHAIN_ID }}/haqq_53211-1/g' | sed 's/{{ VAL_IP }}/10.10.20.18/g' | sed 's/{{ VAL_N }}/01/g' > /etc/tmkms/tmkms01/tmkms.toml
```

### Run TMKMS

```sh
tmkms start -c /etc/tmkms/tmkms01/tmkms.toml >> /var/log/tmkms/tmkms01.log &
```

### Clean TMKMS

```sh
kill `ps -aux | grep tmkms | sed 's/   / /g' | cut -d' ' -f2` && \
rm -rf /etc/tmkms/tmkms*/state/haqq_53211-1-consensus.json
```

## Build genesis

::: tip

**On valop.**

:::
### Install requirements

```sh
apt install bc jq
```

### Prepare init_all script

::: tip

Run these commands on TMKMS host and put results at **init_all.sh** script on valop.

:::

```sh
haqqd tendermint show-validator --home ~/val01
```

```sh
haqqd tendermint show-node-id --home ~/val01
```

### Init default directory

```sh
haqqd init valop --chain-id haqq_53211-1 --keyring-backend os
```

### Create keys

```sh
haqqd keys add val01 --keyring-backend os
```

### Run sctipt

```sh
./haqq_scripts/init_all.sh
```

## Generate node ids

::: tip

**On TMKMS host.**

:::

### Prepare

```sh
mkdir -p ~/nodeids/
```

### Generate keys

```sh
haqqd init sentry01 --home ~/nodeids/sentry01 --chain-id haqq_53211-1 && \
haqqd init sentry02 --home ~/nodeids/sentry02 --chain-id haqq_53211-1 && \
haqqd init sentry03 --home ~/nodeids/sentry03 --chain-id haqq_53211-1 && \
haqqd init frontnode01 --home ~/nodeids/frontnode01 --chain-id haqq_53211-1
```

### Print keys

```sh
cat /dev/null > ~/nodeids/all.txt && \
echo val01 `haqqd tendermint show-node-id --home ~/val01` `cat ~/val01/config/node_key.json` >> ~/nodeids/all.txt && \
echo sentry01 `haqqd tendermint show-node-id --home ~/nodeids/sentry01` `cat ~/nodeids/sentry01/config/node_key.json` >> ~/nodeids/all.txt && \
echo sentry02 `haqqd tendermint show-node-id --home ~/nodeids/sentry02` `cat ~/nodeids/sentry02/config/node_key.json` >> ~/nodeids/all.txt && \
echo sentry03 `haqqd tendermint show-node-id --home ~/nodeids/sentry03` `cat ~/nodeids/sentry03/config/node_key.json` >> ~/nodeids/all.txt && \
echo frontnode01 `haqqd tendermint show-node-id --home ~/nodeids/frontnode01` `cat ~/nodeids/frontnode01/config/node_key.json` >> ~/nodeids/all.txt && \
cat ~/nodeids/all.txt
```