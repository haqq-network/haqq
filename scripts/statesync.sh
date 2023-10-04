#!/bin/bash
# microtick and bitcanna contributed significantly here.
# Pebbledb state sync script.
set -uxe

# Set Golang environment variables.
export GOPATH=~/go
export PATH=$PATH:~/go/bin

# Install with pebbledb 
#go mod edit -replace github.com/tendermint/tm-db=github.com/notional-labs/tm-db@136c7b6
#go mod tidy
#go install -ldflags '-w -s -X github.com/cosmos/cosmos-sdk/types.DBBackend=pebbledb' -tags pebbledb ./...

# NOTE: ABOVE YOU CAN USE ALTERNATIVE DATABASES, HERE ARE THE EXACT COMMANDS
# go install -ldflags '-w -s -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb' -tags rocksdb ./...
# go install -ldflags '-w -s -X github.com/cosmos/cosmos-sdk/types.DBBackend=badgerdb' -tags badgerdb ./...
# go install -ldflags '-w -s -X github.com/cosmos/cosmos-sdk/types.DBBackend=boltdb' -tags boltdb ./...
# Initialize chain.
haqqd init test --chain-id haqq_11235-1

# Get Genesis
wget https://raw.githubusercontent.com/haqq-network/mainnet/master/genesis.json
mv genesis.json ~/.haqqd/config/

wget -O ~/.haqqd/config/adrbook.json https://raw.githubusercontent.com/haqq-network/mainnet/master/addrbook.json

# Get "trust_hash" and "trust_height".
INTERVAL=1000
LATEST_HEIGHT=$(curl -s https://m-s1-tm.haqq.sh:443/block | jq -r .result.block.header.height)
BLOCK_HEIGHT=$(($LATEST_HEIGHT-$INTERVAL)) 
TRUST_HASH=$(curl -s "https://m-s1-tm.haqq.sh:443/block?height=$BLOCK_HEIGHT" | jq -r .result.block_id.hash)

# Print out block and transaction hash from which to sync state.
echo "trust_height: $BLOCK_HEIGHT"
echo "trust_hash: $TRUST_HASH"

# Export state sync variables.
export HAQQD_STATESYNC_ENABLE=true
export HAQQD_P2P_MAX_NUM_INBOUND_PEERS=200
export HAQQD_P2P_MAX_NUM_OUTBOUND_PEERS=200
export HAQQD_STATESYNC_RPC_SERVERS="https://m-s1-tm.haqq.sh:443,https://rpc.tm.haqq.network:443"
export HAQQD_STATESYNC_TRUST_HEIGHT=$BLOCK_HEIGHT
export HAQQD_STATESYNC_TRUST_HASH=$TRUST_HASH

# Fetch and set list of seeds from chain registry.
export HAQQD_P2P_SEEDS=$(curl -s https://raw.githubusercontent.com/cosmos/chain-registry/master/haqq/chain.json | jq -r '[foreach .peers.seeds[] as $item (""; "\($item.id)@\($item.address)")] | join(",")')

# Start chain.
# Add the flag --db_backend=pebbledb if you want to use pebble.

haqqd start --x-crisis-skip-assert-invariants
