<!--
order: 3
-->

# Validator Security

Learn about sentry nodes and HSMs to secure a validator {synopsis}

Each validator candidate is encouraged to run its operations independently, as diverse setups increase the resilience of the network. Validator candidates should commence their setup phase now in order to be on time for launch.

## Horcrux

Horcrux is a [multi-party-computation (MPC)](https://en.wikipedia.org/wiki/Secure_multi-party_computation) signing service for Tendermint nodes

> Take your validator infrastructure to the next level of security and availability:
>
> - Composed of a cluster of signer nodes in place of the [remote signer](https://docs.tendermint.com/master/nodes/remote-signer.html), enabling High Availability (HA) for block signing through fault tolerance.
> - Secure your validator private key by splitting it across multiple private signer nodes using threshold Ed25519 signatures
> - Add security and availability without sacrificing block sign performance.

See documentation [here](https://github.com/strangelove-ventures/horcrux/blob/main/docs/migrating.md) to learn how to upgrade your validator infrastructure with Horcrux.

## Tendermint KMS

[Tendermint KMS](../kms/kms.md) is a signature service with support for Hardware Security Modules (HSMs), such as YubiHSM2 and Ledger Nano. It’s intended to be run alongside Haqq Validators, ideally on separate physical hosts, providing defense-in-depth for online validator signing keys, double signing protection, and functioning as a central signing service that can be used when operating multiple validators in several Cosmos Zones.

For more details, please see [Tendermint KMS](../kms/kms.md)

## Hardware HSM

It is mission critical that an attacker cannot steal a validator's key. If this is possible, it puts the entire stake delegated to the compromised validator at risk. Hardware security modules are an important strategy for mitigating this risk.

HSM modules must support `ed25519` signatures for Haqq. The [YubiHSM 2](https://www.yubico.com/products/hardware-security-module/) supports `ed25519` and can be used with this YubiKey [library](https://github.com/iqlusioninc/yubihsm.rs).

::: danger
🚨 **IMPORTANT**: The YubiHSM can protect a private key but **cannot ensure** in a secure setting that it won't sign the same block twice.
:::

## Sentry Nodes

Validators are responsible for ensuring that the network can sustain denial of service attacks.

The aim of a Sentry node architecture is to ensure Validator nodes are not exposed on a public network. Validators are responsible for ensuring that the network can sustain denial of service attacks.

One recommended way to mitigate these risks is for validators to carefully structure their network topology in a so-called sentry node architecture.

Validator nodes should only connect to full-nodes they trust because they operate them themselves or are run by other validators they know socially. A validator node will typically run in a data center. Most data centers provide direct links the networks of major cloud providers. The validator can use those links to connect to sentry nodes in the cloud. This shifts the burden of denial-of-service from the validator's node directly to its sentry nodes, and may require new sentry nodes be spun up or activated to mitigate attacks on existing ones.

Sentry nodes can be quickly spun up or change their IP addresses. Because the links to the sentry nodes are in private IP space, an internet based attacked cannot disturb them directly. This will ensure validator block proposals and votes always make it to the rest of the network.

::: tip
Read more about Sentry Nodes on the [forum](https://forum.cosmos.network/t/sentry-node-architecture-overview/454)
:::

## Environment Variables

Note that while explicit command-line flags will take precedence over environment variables and configuration files. For this reason, it's imperative that lock down your environment such that any critical parameters are defined as flags on the binary or prevent modification of any environment variables.

## Setting up a Validator

When setting up a validator there are countless ways to configure your setup. This guide is aimed at showing one of them, the sentry node design. This design is mainly for DDOS prevention.

### Network Layout

![Network Layout](./img/sentry_haqq.svg)

The diagram is based on AWS, other cloud providers will have similar solutions to design a solution. Running nodes is not limited to cloud providers, you can run nodes on bare metal systems as well. The architecture will be the same no matter which setup you decide to go with.

The proposed network diagram is similar to the classical backend/frontend separation of services in a corporate environment. The “backend” in this case is the private network of the validator in the data center. The data center network might involve multiple subnets, firewalls and redundancy devices, which is not detailed on this diagram. The important point is that the data center allows direct connectivity to the chosen cloud environment. Amazon AWS has “Direct Connect”, while Google Cloud has “Partner Interconnect”. This is a dedicated connection to the cloud provider (usually directly to your virtual private cloud instance in one of the regions).

All sentry nodes (the “frontend”) connect to the validator using this private connection. The validator does not have a public IP address to provide its services.

Amazon has multiple availability zones within a region. One can install sentry nodes in other regions too. In this case the second, third and further regions need to have a private connection to the validator node. This can be achieved by VPC Peering (“VPC Network Peering” in Google Cloud). In this case, the second, third and further region sentry nodes will be directed to the first region and through the direct connect to the data center, arriving to the validator.

A more persistent solution (not detailed on the diagram) is to have multiple direct connections to different regions from the data center. This way VPC Peering is not mandatory, although still beneficial for the sentry nodes. This overcomes the risk of depending on one region. It is more costly.

### Local Configuration

![Local Configuration](./img/local_sentry.svg)

The validator will only talk to the sentry that are provided, the sentry nodes will communicate to the validator via a secret connection and the rest of the network through a normal connection. The sentry nodes do have the option of communicating with each other as well.

When initializing nodes there are five parameters in the `config.toml` that may need to be altered.

- `pex:` boolean. This turns the peer exchange reactor on or off for a node. When `pex=false`, only the `persistent_peers` list is available for connection.
- `persistent_peers:` a comma separated list of `nodeID@ip:port` values that define a list of peers that are expected to be online at all times. This is necessary at first startup because by setting `pex=false` the node will not be able to join the network.
- `unconditional_peer_ids:` comma separated list of nodeID's. These nodes will be connected to no matter the limits of inbound and outbound peers. This is useful for when sentry nodes have full address books.
- `private_peer_ids:` comma separated list of nodeID's. These nodes will not be gossiped to the network. This is an important field as you do not want your validator IP gossiped to the network.
- `addr_book_strict:` boolean. By default nodes with a routable address will be considered for connection. If this setting is turned off (false), non-routable IP addresses, like addresses in a private network can be added to the address book.
- `double_sign_check_height` int64 height.  How many blocks to look back to check existence of the node's consensus votes before joining consensus When non-zero, the node will panic upon restart if the same consensus key was used to sign {double_sign_check_height} last blocks. So, validators should stop the state machine, wait for some blocks, and then restart the state machine to avoid panic.

#### Validator Node Configuration

| Config Option             | Setting                     |
| ------------------------  | --------------------------- |
| `pex`                     | `false`                     |
| `persistent_peers`        | `list of sentry nodes`      |
| `private_peer_ids`        | `none`                      |
| `unconditional_peer_ids`  | `optionally sentry node IDs`|
| `addr_book_strict`        | `false`                     |
| `double_sign_check_height`| `10`                        |

The validator node should have `pex=false` so it does not gossip to the entire network. The persistent peers will be your sentry nodes. Private peers can be left empty as the validator is not trying to hide who it is communicating with. Setting unconditional peers is optional for a validator because they will not have a full address books.

#### Sentry Node Configuration

| Config Option             | Setting                     |
| ------------------------  | --------------------------- |
| `pex`                     | `true`                      |
| `persistent_peers`        | `optionally`                |
| `private_peer_ids`        | `validator node ID`         |
| `unconditional_peer_ids`  | `validator node ID, optionally sentry node IDs`|
| `addr_book_strict`        | `false`                     |

The sentry nodes should be able to talk to the entire network hence why `pex=true`. The persistent peers of a sentry node will be the validator, and optionally other sentry nodes. The sentry nodes should make sure that they do not gossip the validator's ip, to do this you must put the validators nodeID as a private peer. The unconditional peer IDs will be the validator ID and optionally other sentry nodes.

::: tip
Do not forget to secure your node's firewalls when setting them up.
:::

### Validator keys

Protecting a validator's consensus key is the most important factor to take in when designing your setup. The key that a validator is given upon creation of the node is called a consensus key, it has to be online at all times in order to vote on blocks. It is **not recommended** to merely hold your private key in the default json file (`priv_validator_key.json`). Fortunately, the [Interchain Foundation](https://interchain.io) has worked with a team to build a key management server for validators. You can find documentation on how to use it [here](https://github.com/iqlusioninc/tmkms), it is used extensively in production. You are not limited to using this tool, there are also [HSMs](https://safenet.gemalto.com/data-encryption/hardware-security-modules-hsms/), there is not a recommended HSM.

Currently Tendermint uses [Ed25519](https://ed25519.cr.yp.to/) keys which are widely supported across the security sector and HSMs.

### Common attacks for a validator node

Distributed denial of service (DDoS) attack will halt the vote messages between validators and prevent blocks from being committed.

**Compromise of keys**

The most valuable asset for a validator is the reputation it has with other validators. If an attacker gains control of or access to the validator, then they can control anything that is signed with the keys. So an attacker who compromises the keys has full control over all the signatures in the blockchain. Even if keys are secured, any copies of them in backups or on other support systems could still be compromised and used to clone the validator to malicious ends.

**Trusted Link** 

Validators may be able to be compromised by attackers if they have access to the system logging in. These trusted communication links will typically be that to the validator from the sysadmin desktop, but also from other services, such as a password reset for an email address on record with the validator system. If an attacker has access to that system, they can at a minimum piggyback on that remote session.

**Tendermint Network Vulnerability**

Vulnerability of a Tendermint network services has two important implications. The first is that it is potentially vulnerable to attacks by third party. An attacker could attempt to generate colliding transactions and blocks, so as to disrupt the network. To protect the safety of validator node, one common solution is to setup sentry nodes. A sentry node is just a full node, which could be used to protect validator node from DDoS attack by constantly relaying the validator's signed messages to public network. In this way, a flood could be mitigated.
