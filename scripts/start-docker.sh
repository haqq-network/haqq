#!/bin/bash

KEY="mykey"
CHAINID="haqq_121799-1"
MONIKER="mymoniker"
DATA_DIR=$(mktemp -d -t haqq-datadir.XXXXX)

echo "create and add new keys"
./haqqd keys add $KEY --home $DATA_DIR --no-backup --chain-id $CHAINID --algo "eth_secp256k1" --keyring-backend test
echo "init Evmos with moniker=$MONIKER and chain-id=$CHAINID"
./haqqd init $MONIKER --chain-id $CHAINID --home $DATA_DIR
echo "prepare genesis: Allocate genesis accounts"
./haqqd add-genesis-account \
"$(./haqqd keys show $KEY -a --home $DATA_DIR --keyring-backend test)" 1000000000000000000aphoton,1000000000000000000stake \
--home $DATA_DIR --keyring-backend test
echo "prepare genesis: Sign genesis transaction"
./haqqd gentx $KEY 1000000000000000000stake --keyring-backend test --home $DATA_DIR --keyring-backend test --chain-id $CHAINID
echo "prepare genesis: Collect genesis tx"
./haqqd collect-gentxs --home $DATA_DIR
echo "prepare genesis: Run validate-genesis to ensure everything worked and that the genesis file is setup correctly"
./haqqd validate-genesis --home $DATA_DIR

echo "starting haqq node $i in background ..."
./haqqd start --pruning=nothing --rpc.unsafe \
--keyring-backend test --home $DATA_DIR \
>$DATA_DIR/node.log 2>&1 & disown

echo "started haqq node"
tail -f /dev/null