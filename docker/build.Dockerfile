### build stage
FROM golang:1.23-bookworm AS build-env

WORKDIR /go/src/github.com/haqq-network/haqq

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN make build

RUN go install github.com/MinseokOh/toml-cli@latest
RUN go install github.com/cosmos/cosmos-sdk/cosmovisor/cmd/cosmovisor@latest

### run stage
FROM alpine:3.18

WORKDIR /root

COPY --from=build-env /go/bin/toml-cli /usr/bin/toml-cli
COPY --from=build-env /go/bin/cosmovisor /usr/bin/cosmovisor
COPY --from=build-env /go/src/github.com/haqq-network/haqq/build/haqqd /usr/bin/haqqd

RUN apk add --no-cache \
    ca-certificates jq \
    curl bash \
    vim lz4 \
    tini \
    gcompat
    
RUN addgroup -g 1000 haqq \
    && adduser -S -h /home/haqq -D haqq -u 1000 -G haqq

USER 1000
WORKDIR /home/haqq

ENV DAEMON_NAME=haqqd
ENV DAEMON_HOME=/home/haqq/.haqqd
ENV DAEMON_ALLOW_DOWNLOAD_BINARIES=true
ENV DAEMON_RESTART_AFTER_UPGRADE=true
ENV UNSAFE_SKIP_BACKUP=false

RUN mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin $DAEMON_HOME/cosmovisor/upgrades \
    && cp /usr/bin/haqqd $DAEMON_HOME/cosmovisor/genesis/bin/haqqd

EXPOSE 26656 26657 1317 9090 8545 8546

ENTRYPOINT ["/sbin/tini", "--"]

CMD ["cosmovisor"]
