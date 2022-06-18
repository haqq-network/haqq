
KEY="localkey"
CHAINID="haqq_121799-1"
MONIKER="localtestnet"
KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# to trace evm
#TRACE="--trace"
TRACE=""

# validate dependencies are installed
command -v jq > /dev/null 2>&1 || { echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"; exit 1; }

# remove existing daemon
rm -rf ~/.haqqd*

# remove previus builds
rm -rf build

make 
BINARY=build/haqqd

$BINARY config keyring-backend $KEYRING
$BINARY config chain-id $CHAINID

# if $KEY exists it should be deleted
$BINARY keys add $KEY --keyring-backend $KEYRING

# Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
$BINARY init $MONIKER --chain-id $CHAINID 

# Change parameter token denominations to aISLM
cat $HOME/.haqqd/config/genesis.json | jq '.app_state["staking"]["params"]["bond_denom"]="aISLM"' > $HOME/.haqqd/config/tmp_genesis.json && mv $HOME/.haqqd/config/tmp_genesis.json $HOME/.haqqd/config/genesis.json
cat $HOME/.haqqd/config/genesis.json | jq '.app_state["crisis"]["constant_fee"]["denom"]="aISLM"' > $HOME/.haqqd/config/tmp_genesis.json && mv $HOME/.haqqd/config/tmp_genesis.json $HOME/.haqqd/config/genesis.json
cat $HOME/.haqqd/config/genesis.json | jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="aISLM"' > $HOME/.haqqd/config/tmp_genesis.json && mv $HOME/.haqqd/config/tmp_genesis.json $HOME/.haqqd/config/genesis.json
cat $HOME/.haqqd/config/genesis.json | jq '.app_state["mint"]["params"]["mint_denom"]="aISLM"' > $HOME/.haqqd/config/tmp_genesis.json && mv $HOME/.haqqd/config/tmp_genesis.json $HOME/.haqqd/config/genesis.json
cat $HOME/.haqqd/config/genesis.json | jq '.app_state["evm"]["params"]["evm_denom"]="aISLM"' > $HOME/.haqqd/config/tmp_genesis.json && mv $HOME/.haqqd/config/tmp_genesis.json $HOME/.haqqd/config/genesis.json

# Set gas limit in genesis
cat $HOME/.haqqd/config/genesis.json | jq '.consensus_params["block"]["max_gas"]="10000000"' > $HOME/.haqqd/config/tmp_genesis.json && mv $HOME/.haqqd/config/tmp_genesis.json $HOME/.haqqd/config/genesis.json

# 1 min for proposal's vote vaiting
cat $HOME/.haqqd/config/genesis.json | jq '.consensus_params["gov"]["voting_params"]["voting_period"]="60s"' > $HOME/.haqqd/config/tmp_genesis.json && mv $HOME/.haqqd/config/tmp_genesis.json $HOME/.haqqd/config/genesis.json

# if you need disable produce empty block, uncomment code below
# if [[ "$OSTYPE" == "darwin"* ]]; then
#     sed -i '' 's/create_empty_blocks = true/create_empty_blocks = false/g' $HOME/.haqqd/config/config.toml
#   else
#     sed -i 's/create_empty_blocks = true/create_empty_blocks = false/g' $HOME/.haqqd/config/config.toml
# fi

if [[ $1 == "pending" ]]; then
  if [[ "$OSTYPE" == "darwin"* ]]; then
      sed -i '' 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "30s"/g' $HOME/.haqqd/config/config.toml
      sed -i '' 's/timeout_propose = "3s"/timeout_propose = "30s"/g' $HOME/.haqqd/config/config.toml
      sed -i '' 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' $HOME/.haqqd/config/config.toml
      sed -i '' 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' $HOME/.haqqd/config/config.toml
      sed -i '' 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' $HOME/.haqqd/config/config.toml
      sed -i '' 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' $HOME/.haqqd/config/config.toml
      sed -i '' 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' $HOME/.haqqd/config/config.toml
      sed -i '' 's/timeout_commit = "5s"/timeout_commit = "150s"/g' $HOME/.haqqd/config/config.toml
      sed -i '' 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' $HOME/.haqqd/config/config.toml
  else
      sed -i 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "30s"/g' $HOME/.haqqd/config/config.toml
      sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' $HOME/.haqqd/config/config.toml
      sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' $HOME/.haqqd/config/config.toml
      sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' $HOME/.haqqd/config/config.toml
      sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' $HOME/.haqqd/config/config.toml
      sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' $HOME/.haqqd/config/config.toml
      sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' $HOME/.haqqd/config/config.toml
      sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' $HOME/.haqqd/config/config.toml
      sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' $HOME/.haqqd/config/config.toml
  fi
fi

# Allocate genesis accounts (cosmos formatted addresses)
$BINARY add-genesis-account $KEY 100000000000000000000000000aISLM --keyring-backend $KEYRING &> /dev/null

# Sign genesis transaction
$BINARY gentx $KEY 1000000000000000000000aISLM --keyring-backend $KEYRING --chain-id $CHAINID &> /dev/null

# Collect genesis tx
$BINARY collect-gentxs &> /dev/null

# Run this to ensure everything worked and that the genesis file is setup correctly
$BINARY validate-genesis 

if [[ $1 == "pending" ]]; then
  echo "pending mode is on, please wait for the first block committed."
fi

echo "\n### Priv key for Metamask ####"
$BINARY keys unsafe-export-eth-key $KEY --home=$HOME/.haqqd --keyring-backend $KEYRING
echo "\n\n\n"

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
$BINARY start --pruning=nothing $TRACE --log_level $LOGLEVEL \
--minimum-gas-prices=0.0001aISLM \
--json-rpc.api eth,txpool,personal,net,debug,web3 \
--json-rpc.enable true --keyring-backend $KEYRING