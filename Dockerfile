### build stage
FROM golang:1.20.4-bullseye AS build-env

WORKDIR /go/src/github.com/haqq-network/haqq

COPY . .

ENV CGO_ENABLED=0
RUN make build

### run stage
FROM golang:1.20.4-bullseye

RUN apt-get update  \ 
&& apt-get install ca-certificates jq=1.6-2.1 -y --no-install-recommends

WORKDIR /root

COPY --from=build-env /go/src/github.com/haqq-network/haqq/build/haqqd /usr/bin/haqqd

RUN mkdir -p /home/coin

EXPOSE 26656 26657 1317 9090 8545 8546

CMD ["haqqd"]
