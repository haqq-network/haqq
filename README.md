<!--
parent:
  order: false
-->

<div align="center">
  <h1> Haqq </h1>
</div>

<!-- TODO: add banner -->
<!-- ![banner](docs/ethermint.jpg) -->

Haqq is a scalable, high-throughput Proof-of-Stake blockchain that is fully compatible and interoperable with Ethereum. 
It's built using the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk) which runs on top of [Tendermint Core](https://github.com/tendermint/tendermint) consensus engine.
Ethereum compatibility allows developers to build applications on Haqq using the existing Ethereum codebase and toolset,
without rewriting smart contracts that already work on Ethereum or other Ethereum-compatible networks.
Ethereum compatibility is done using modules built by [Tharsis](https://thars.is) for their [Evmos](https://evmos.org) network.

Haqq Network’s Shariah-compliant native currency is ISLM – [Fatwa](https://islamiccoin.net/fatwa).

**Note**: Requires [Go 1.18+](https://golang.org/dl)

## Installation

**Note**: Make sure that you install `Go` (you can follow official guide [Install Go](https://go.dev/doc/install) and `GOPATH` is configured for source directory).

```bash
cd $HOME && \
ver="1.19.1" && \
wget "https://golang.org/dl/go$ver.linux-amd64.tar.gz" && \
sudo rm -rf /usr/local/go && \
sudo tar -C /usr/local -xzf "go$ver.linux-amd64.tar.gz" && \
rm "go$ver.linux-amd64.tar.gz" && \
echo "export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin" >> $HOME/.bash_profile && \
source $HOME/.bash_profile && \
go version
```

For prerequisites and detailed build instructions please read the [Installation](https://docs.haqq.network/quickstart/installation.html) instructions. Once the dependencies are installed, run:

```bash
make install
```

Or check out the latest [release](https://github.com/haqq-network/haqq/releases).

## Quick Start

To learn how the Haqq works from a high-level perspective, go to the [Introduction](https://docs.haqq.network/intro/overview.html) section from the documentation. You can also check the instructions to [Run a Node](https://docs.haqq.network/guides/localnet/single_node.html).

<!-- ## Community -->
<!-- ## Contributing -->
