#!/bin/bash
set -euo pipefail

NODE_NUM=2 # Number of nodes with 0 indexing
DEFAULT_DENOM="aISLM"
CHAIN_ID="haqq_121799-1"
MONIKER_PREFIX="validator"
KEYRING="test"
HAQQD_BIN="haqqd"
LOCALNET_PATH="${PWD}/localnet"

command -v jq > /dev/null 2>&1 || { echo >&2 "jq not installed."; exit 1; }
command -v $HAQQD_BIN > /dev/null 2>&1 || { echo >&2 "$HAQQD_BIN not found in PATH."; exit 1; }

# Clean up
rm -rf "$LOCALNET_PATH"

for i in $(seq 0 $NODE_NUM); do
    NODE_HOME="$LOCALNET_PATH/node$i"
    rm -rf "$NODE_HOME"
    mkdir -p "$NODE_HOME"

    $HAQQD_BIN init "$MONIKER_PREFIX$i" --chain-id $CHAIN_ID --home "$NODE_HOME"

    $HAQQD_BIN config keyring-backend $KEYRING --home "$NODE_HOME"
    $HAQQD_BIN config chain-id $CHAIN_ID --home "$NODE_HOME"
done

# Build peer lists using container hostnames haqq_localnet_nodeX
# Collect node IDs
declare -A NODE_IDS
for i in $(seq 0 $NODE_NUM); do
    NODE_HOME="$LOCALNET_PATH/node$i"
    NODE_IDS[$i]="$($HAQQD_BIN tendermint show-node-id --home "$NODE_HOME")"
done

# Update seeds and persistent_peers for each node
for i in $(seq 0 $NODE_NUM); do
    CONFIG="$LOCALNET_PATH/node$i/config/config.toml"
    PEERS=""
    for j in $(seq 0 $NODE_NUM); do
        if [ $j -ne $i ]; then
            ID_J="${NODE_IDS[$j]}"
            HOST_J="haqq_localnet_node$j"
            ENTRY="$ID_J@$HOST_J:26656"
            if [ -z "$PEERS" ]; then
                PEERS="$ENTRY"
            else
                PEERS="$PEERS,$ENTRY"
            fi
        fi
    done
    # Clear seeds and set persistent peers
    sed -i 's/^seeds = .*/seeds = ""/' "$CONFIG"
    sed -i "s/^persistent_peers = .*/persistent_peers = \"$PEERS\"/" "$CONFIG"
done

echo "Creating and verifying keys..."
for i in $(seq 0 $NODE_NUM); do
    NODE_HOME="$LOCALNET_PATH/node$i"
    KEY_NAME="$MONIKER_PREFIX$i"
    (echo -n || true) | $HAQQD_BIN keys add "$KEY_NAME" --keyring-backend $KEYRING --home "$NODE_HOME" > "$NODE_HOME/key_output.txt" 2>&1 || { echo "Failed to add key $KEY_NAME for $NODE_HOME"; cat "$NODE_HOME/key_output.txt"; exit 1; }
    ADDR=$($HAQQD_BIN keys show "$KEY_NAME" --keyring-backend $KEYRING --home "$NODE_HOME" -a 2>/dev/null)
    if [ -z "$ADDR" ]; then
        echo "Failed to obtain address for $KEY_NAME at $NODE_HOME"; cat "$NODE_HOME/key_output.txt"; exit 1
    fi
    echo "Node $i ($KEY_NAME) address: $ADDR"
done

# Allocate genesis accounts for each planned validator. This must be done BEFORE any gentx is created.
for i in $(seq 0 $NODE_NUM); do
    NODE_HOME="$LOCALNET_PATH/node$i"
    KEY_NAME="$MONIKER_PREFIX$i"
    ADDR=$($HAQQD_BIN keys show "$KEY_NAME" --keyring-backend $KEYRING --home "$NODE_HOME" -a)
    $HAQQD_BIN add-genesis-account "$ADDR" 10000000000000000000$DEFAULT_DENOM --keyring-backend $KEYRING --home "$LOCALNET_PATH/node0"
    echo "Genesis funded: $ADDR -> 10000000000000000000$DEFAULT_DENOM"
done

# Apply genesis param changes only once (node0)
GENESIS_PATH="$LOCALNET_PATH/node0/config/genesis.json"
tmp() { echo "$GENESIS_PATH" | sed 's/genesis.json/tmp_genesis.json/'; }
cat $GENESIS_PATH | jq '.app_state["staking"]["params"]["bond_denom"]="aISLM"' > $(tmp) && mv $(tmp) $GENESIS_PATH
cat $GENESIS_PATH | jq '.app_state["crisis"]["constant_fee"]["denom"]="aISLM"' > $(tmp) && mv $(tmp) $GENESIS_PATH
cat $GENESIS_PATH | jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="aISLM"' > $(tmp) && mv $(tmp) $GENESIS_PATH
cat $GENESIS_PATH | jq '.app_state["mint"]["params"]["mint_denom"]="aISLM"' > $(tmp) && mv $(tmp) $GENESIS_PATH
cat $GENESIS_PATH | jq '.app_state["evm"]["params"]["evm_denom"]="aISLM"' > $(tmp) && mv $(tmp) $GENESIS_PATH
# voting period
cat $GENESIS_PATH | jq '.app_state["gov"]["params"]["voting_period"]="120s"' > $(tmp) && mv $(tmp) $GENESIS_PATH
cat $GENESIS_PATH | jq '.app_state["gov"]["params"]["expedited_voting_period"]="100s"' > $(tmp) && mv $(tmp) $GENESIS_PATH

# block gas
cat $GENESIS_PATH | jq '.consensus_params["block"]["max_gas"]="10000000"' > $(tmp) && mv $(tmp) $GENESIS_PATH
# distribution
cat $GENESIS_PATH | jq '.app_state["distribution"]["params"]["base_proposer_reward"]="0.010000000000000000"' > $(tmp) && mv $(tmp) $GENESIS_PATH
cat $GENESIS_PATH | jq '.app_state["distribution"]["params"]["bonus_proposer_reward"]="0.040000000000000000"' > $(tmp) && mv $(tmp) $GENESIS_PATH
cat $GENESIS_PATH | jq '.app_state["distribution"]["params"]["community_tax"]="0.100000000000000000"' > $(tmp) && mv $(tmp) $GENESIS_PATH

# Synchronize funded/modified genesis to all other nodes BEFORE gentx
for i in $(seq 1 $NODE_NUM); do
    cp $GENESIS_PATH "$LOCALNET_PATH/node$i/config/genesis.json"
done

# Ensure gentx dir exists before creating gentx files
mkdir -p "$LOCALNET_PATH/gentx"
# Also ensure node0's expected gentx directory exists for collect-gentxs
mkdir -p "$LOCALNET_PATH/node0/config/gentx"

echo "Generating gentx files..."
# Collect gents and gentx
for i in $(seq 0 $NODE_NUM); do
    NODE_HOME="$LOCALNET_PATH/node$i"
    KEY_NAME="$MONIKER_PREFIX$i"
    NODE_ID_I="${NODE_IDS[$i]}"
    HOST_I="haqq_localnet_node$i"
    if [ $i -eq 0 ]; then
        AMOUNT="2000000000000000000$DEFAULT_DENOM"
    else
        AMOUNT="1000000000000000000$DEFAULT_DENOM"
    fi
    $HAQQD_BIN gentx "$KEY_NAME" $AMOUNT \
        --keyring-backend $KEYRING \
        --chain-id $CHAIN_ID \
        --home "$NODE_HOME" \
        --node-id "$NODE_ID_I" \
        --ip "$HOST_I" \
        --output-document "$LOCALNET_PATH/gentx/node$i.json"
done

# Move all gentx to node0/config/gentx
for i in $(seq 0 $NODE_NUM); do
    cp "$LOCALNET_PATH/gentx/node$i.json" "$LOCALNET_PATH/node0/config/gentx/gentx-$(printf "%02d" $i).json"
done

touch "$LOCALNET_PATH/gentx/README"

$HAQQD_BIN collect-gentxs --home "$LOCALNET_PATH/node0"

echo "Propagating updated genesis to all nodes..."
for i in $(seq 1 $NODE_NUM); do
    cp $GENESIS_PATH "$LOCALNET_PATH/node$i/config/genesis.json"
done

echo "Multi-node haqq localnet configured at $LOCALNET_PATH. To start a node, run:"
echo "$HAQQD_BIN start --home $LOCALNET_PATH/nodeX --pruning=nothing --json-rpc.enable true --keyring-backend $KEYRING"

