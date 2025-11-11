docker cp ~/haqq-project/v1.10.0/haqqd "haqq_localnet_node0:/app/.haqqd/cosmovisor/upgrades/v1.10.0/bin/haqqd"
docker cp ~/haqq-project/v1.10.0/haqqd "haqq_localnet_node1:/app/.haqqd/cosmovisor/upgrades/v1.10.0/bin/haqqd"
docker cp ~/haqq-project/v1.10.0/haqqd "haqq_localnet_node2:/app/.haqqd/cosmovisor/upgrades/v1.10.0/bin/haqqd"

docker cp ~/haqq-project/proposal.json "haqq_localnet_node0:/app/proposal.json"

haqqd tx gov submit-proposal proposal.json --from validator0 --chain-id haqq_121799-1 --keyring-backend test -y --gas-prices 13939841aISLM --gas 20000000

haqqd tx gov vote 1 yes --from validator0 --yes --keyring-backend test --gas-prices 13939841aISLM
haqqd tx gov vote 1 yes --from validator1 --yes --keyring-backend test --gas-prices 13939841aISLM
haqqd tx gov vote 1 yes --from validator2 --yes --keyring-backend test --gas-prices 13939841aISLM

