FROM golang:1.19-alpine3.17 AS build-env

RUN apk add --no-cache --update \
  ca-certificates \
  git \
  gcc \
  make

WORKDIR /go/src/github.com/haqq-network/haqq

COPY . .
ENV CGO_ENABLED=0

RUN make build

FROM alpine:3.17

RUN apk update
RUN apk add ca-certificates jq

WORKDIR /root

COPY --from=build-env /go/src/github.com/haqq-network/haqq/build/haqqd /usr/bin/haqqd
COPY scripts/start_node.sh /

RUN mkdir -p /home/coin

EXPOSE 26656 26657 1317 9090 8545

ENTRYPOINT [ "/start_node.sh" ]
