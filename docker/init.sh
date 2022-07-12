#!/bin/sh

KEY="localkey"
CHAINID="haqq_121799-1"
MONIKER="localtestnet"
KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# to trace evm
#TRACE="--trace"
TRACE=""

if [ -d "root/.haqqd/config" ]
then
  echo "Node config directory already exist!"
else

# validate dependencies are installed
command -v jq > /dev/null 2>&1 || { echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"; exit 1 ;}

haqqd config keyring-backend $KEYRING
haqqd config chain-id $CHAINID

# if $KEY exists it should be deleted
haqqd keys add $KEY --keyring-backend $KEYRING

# Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
haqqd init $MONIKER --chain-id $CHAINID 

# Change parameter token denominations to aISLM
cat root/.haqqd/config/genesis.json | jq '.app_state["staking"]["params"]["bond_denom"]="aISLM"' > root/.haqqd/config/tmp_genesis.json && mv root/.haqqd/config/tmp_genesis.json root/.haqqd/config/genesis.json
cat root/.haqqd/config/genesis.json | jq '.app_state["crisis"]["constant_fee"]["denom"]="aISLM"' > root/.haqqd/config/tmp_genesis.json && mv root/.haqqd/config/tmp_genesis.json root/.haqqd/config/genesis.json
cat root/.haqqd/config/genesis.json | jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="aISLM"' > root/.haqqd/config/tmp_genesis.json && mv root/.haqqd/config/tmp_genesis.json root/.haqqd/config/genesis.json
cat root/.haqqd/config/genesis.json | jq '.app_state["mint"]["params"]["mint_denom"]="aISLM"' > root/.haqqd/config/tmp_genesis.json && mv root/.haqqd/config/tmp_genesis.json root/.haqqd/config/genesis.json
cat root/.haqqd/config/genesis.json | jq '.app_state["evm"]["params"]["evm_denom"]="aISLM"' > root/.haqqd/config/tmp_genesis.json && mv root/.haqqd/config/tmp_genesis.json root/.haqqd/config/genesis.json

# Set gas limit in genesis
cat root/.haqqd/config/genesis.json | jq '.consensus_params["block"]["max_gas"]="10000000"' > root/.haqqd/config/tmp_genesis.json && mv root/.haqqd/config/tmp_genesis.json root/.haqqd/config/genesis.json

# 1 min for proposal's vote vaiting
cat root/.haqqd/config/genesis.json | jq '.consensus_params["gov"]["voting_params"]["voting_period"]="60s"' > root/.haqqd/config/tmp_genesis.json && mv root/.haqqd/config/tmp_genesis.json root/.haqqd/config/genesis.json

# if you need disable produce empty block, uncomment code below
# if [[ "$OSTYPE" == "darwin"* ]]; then
#     sed -i '' 's/create_empty_blocks = true/create_empty_blocks = false/g' root/.haqqd/config/config.toml
#   else
#     sed -i 's/create_empty_blocks = true/create_empty_blocks = false/g' root/.haqqd/config/config.toml
# fi

if [[ $1 == "pending" ]]; then
  if [[ "$OSTYPE" == "darwin"* ]]; then
      sed -i '' 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "30s"/g' root/.haqqd/config/config.toml
      sed -i '' 's/timeout_propose = "3s"/timeout_propose = "30s"/g' root/.haqqd/config/config.toml
      sed -i '' 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' root/.haqqd/config/config.toml
      sed -i '' 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' root/.haqqd/config/config.toml
      sed -i '' 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' root/.haqqd/config/config.toml
      sed -i '' 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' root/.haqqd/config/config.toml
      sed -i '' 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' root/.haqqd/config/config.toml
      sed -i '' 's/timeout_commit = "5s"/timeout_commit = "150s"/g' root/.haqqd/config/config.toml
      sed -i '' 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' root/.haqqd/config/config.toml
  else
      sed -i 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "30s"/g' root/.haqqd/config/config.toml
      sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' root/.haqqd/config/config.toml
      sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' root/.haqqd/config/config.toml
      sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' root/.haqqd/config/config.toml
      sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' root/.haqqd/config/config.toml
      sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' root/.haqqd/config/config.toml
      sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' root/.haqqd/config/config.toml
      sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' root/.haqqd/config/config.toml
      sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' root/.haqqd/config/config.toml
  fi
fi

# Allocate genesis accounts (cosmos formatted addresses)
haqqd add-genesis-account $KEY 100000000000000000000000000aISLM --keyring-backend $KEYRING &> /dev/null

# Sign genesis transaction
haqqd gentx $KEY 1000000000000000000000aISLM --keyring-backend $KEYRING --chain-id $CHAINID &> /dev/null

# Collect genesis tx
haqqd collect-gentxs &> /dev/null

# Run this to ensure everything worked and that the genesis file is setup correctly
haqqd validate-genesis 

if [[ $1 == "pending" ]]; then
  echo "pending mode is on, please wait for the first block committed."
fi

echo "\n### Priv key for Metamask ####"
haqqd keys unsafe-export-eth-key $KEY --home=root/.haqqd --keyring-backend $KEYRING
echo "\n\n\n"

fi