#!/bin/sh

KEYRING="test"
LOGLEVEL="info"
# to trace evm
# TRACE="--trace"
TRACE=""
MONIKER="test"

if [ ! -d "root/.haqqd/config" ]
then
  echo "Node config directory doesn't exist!"
else

haqqd start --pruning=nothing $TRACE --log_level $LOGLEVEL \
--minimum-gas-prices=0.0001aISLM \
--json-rpc.api eth,txpool,personal,net,debug,web3 \
--json-rpc.enable true --keyring-backend $KEYRING

fi