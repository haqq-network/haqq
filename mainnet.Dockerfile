ARG from_image

FROM $from_image

ENV CUSTOM_MONIKER="mainnet_seed_node"

RUN [[ ! -f $HOME/.haqqd/config/genesis.json ]] && \
    haqqd config chain-id haqq_11235-1 && \
    haqqd init $CUSTOM_MONIKER --chain-id haqq_11235-1 && \
    curl -OL https://raw.githubusercontent.com/haqq-network/mainnet/master/genesis.json && \
    mv genesis.json $HOME/.haqqd/config/genesis.json && \
    curl -OL https://raw.githubusercontent.com/haqq-network/mainnet/master/addrbook.json && \
    mv addrbook.json $HOME/.haqqd/config/addrbook.json