FROM golang:stretch AS build-env

WORKDIR /go/src/github.com/haqq-network/haqq

RUN apt update
RUN apt install git -y

COPY . .

RUN make build

FROM golang:stretch

RUN apt update
RUN apt install ca-certificates jq -y

WORKDIR /root

COPY --from=build-env /go/src/github.com/haqq-network/haqq/build/haqqd /usr/bin/haqqd
COPY scripts/start_node.sh /

RUN mkdir -p /home/coin

EXPOSE 26656 26657 1317 9090 8545

ENTRYPOINT [ "/start_node.sh" ]
