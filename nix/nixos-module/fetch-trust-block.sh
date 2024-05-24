# fetch trust block from tm rpc url and put into config

set -euo pipefail

echo "Fetching trust block"

LATEST_HEIGHT=$(curl $RPC/status | jq -r '.result.sync_info.latest_block_height')

echo "Latest height: $LATEST_HEIGHT"

TRUST_HEIGHT=$(($LATEST_HEIGHT - $OFFSET))

TRUST_HASH=$(curl $RPC/block?height=$TRUST_HEIGHT | jq -r '.result.block_id.hash')

echo "Trust height: $TRUST_HEIGHT"

TARGET=$(mktemp)
cat $1 | \
    dasel put -r toml -t string -v $TRUST_HASH 'statesync.trust_hash' | \
    dasel put -r toml -t int -v $TRUST_HEIGHT 'statesync.trust_height' > $TARGET

cp $TARGET $1
