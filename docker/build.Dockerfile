### build stage
FROM golang:1.20.6-alpine3.18 AS build-env

WORKDIR /go/src/github.com/haqq-network/haqq

COPY go.mod go.sum ./

RUN set -eux; apk add --no-cache ca-certificates=20230506-r0 build-base=0.5-r3 git=2.40.1-r0 linux-headers=6.3-r0

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
    ca-certificates=20230506-r0 jq=~1.6 \
    curl=~8.4 bash=~5.2 \
    vim=~9.0 lz4=~1.9 \
    tini=~0.19
    
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
