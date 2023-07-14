### build stage
FROM golang:1.20.6-alpine3.18 AS build-env

WORKDIR /go/src/github.com/haqq-network/haqq

COPY go.mod go.sum ./

RUN set -eux; apk add --no-cache ca-certificates=20230506-r0 build-base=0.5-r3 git=2.40.1-r0 linux-headers=6.3-r0

RUN go mod download

COPY . .

RUN make build

RUN go install github.com/MinseokOh/toml-cli@latest

### run stage
FROM alpine:3.18

WORKDIR /root

COPY --from=build-env /go/bin/toml-cli /usr/bin/toml-cli
COPY --from=build-env /go/src/github.com/haqq-network/haqq/build/haqqd /usr/bin/haqqd

RUN apk add --no-cache ca-certificates=20230506-r0 jq=1.6-r3 curl=8.1.2-r0 bash=5.2.15-r5 vim=9.0.1568-r0 lz4=1.9.4-r4 tini=0.19.0-r1 \
    && addgroup -g 1000 haqq \
    && adduser -S -h /home/haqq -D haqq -u 1000 -G haqq

USER 1000
WORKDIR /home/haqq

EXPOSE 26656 26657 1317 9090 8545 8546

ENTRYPOINT ["/sbin/tini", "--"]

CMD ["haqqd"]
