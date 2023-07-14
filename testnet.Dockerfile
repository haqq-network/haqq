ARG from_image

FROM $from_image

ENV CUSTOM_MONIKER="testedge2_seed_node"

RUN [[ ! -f $HOME/.haqqd/config/genesis.json ]] && \
    haqqd config chain-id haqq_54211-3 && \
    haqqd init $CUSTOM_MONIKER --chain-id haqq_54211-3 && \
    curl -OL https://raw.githubusercontent.com/haqq-network/testnets/main/TestEdge2/genesis.tar.bz2 &&\
    bzip2 -d genesis.tar.bz2 && tar -xvf genesis.tar && \
    mv genesis.json $HOME/.haqqd/config/genesis.json && \
    curl -OL https://raw.githubusercontent.com/haqq-network/testnets/main/TestEdge2/addrbook.json && \
    mv addrbook.json $HOME/.haqqd/config/addrbook.json