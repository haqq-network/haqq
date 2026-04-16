#!/usr/bin/env bash
set -euxo pipefail


DEPS_COSMOS_SDK_VERSION=$(cat go.sum | grep -E 'github.com/haqq-network/cosmos-sdk\s' | grep -v -e 'go.mod' | tail -n 1 | awk '{ print $2 }')
DEPS_IBC_GO_VERSION=$(cat go.sum | grep 'github.com/cosmos/ibc-go' | grep -v -e 'go.mod' | tail -n 1 | awk '{ print $2 }')
DEPS_COSMOS_PROTO=$(cat go.sum | grep 'github.com/cosmos/cosmos-proto' | grep -v -e 'go.mod' | tail -n 1 | awk '{ print $2 }')
DEPS_COSMOS_GOGOPROTO=$(cat go.sum | grep 'github.com/cosmos/gogoproto' | grep -v -e 'go.mod' | tail -n 1 | awk '{ print $2 }')
DEPS_COSMOS_ICS23=go/$(cat go.sum | grep 'github.com/cosmos/ics23/go' | grep -v -e 'go.mod' | tail -n 1 | awk '{ print $2 }')

mkdir -p "$THIRD_PARTY_DIR/cosmos_tmp" && \
	cd "$THIRD_PARTY_DIR/cosmos_tmp" && \
	git init && \
	git remote add origin "https://github.com/haqq-network/cosmos-sdk.git" && \
	git config core.sparseCheckout true && \
	printf "proto\nthird_party\n" > .git/info/sparse-checkout && \
	git pull origin "$DEPS_COSMOS_SDK_VERSION" && \
	rm -f ./proto/buf.* && \
	mv ./proto/* ..
rm -rf "$THIRD_PARTY_DIR/cosmos_tmp"

mkdir -p "$THIRD_PARTY_DIR/ibc_tmp" && \
	cd "$THIRD_PARTY_DIR/ibc_tmp" && \
	git init && \
	git remote add origin "https://github.com/cosmos/ibc-go.git" && \
	git config core.sparseCheckout true && \
	printf "proto\n" > .git/info/sparse-checkout && \
	git pull origin "$DEPS_IBC_GO_VERSION" && \
	rm -f ./proto/buf.* && \
	mv ./proto/* ..
rm -rf "$THIRD_PARTY_DIR/ibc_tmp"

mkdir -p "$THIRD_PARTY_DIR/cosmos_proto_tmp" && \
	cd "$THIRD_PARTY_DIR/cosmos_proto_tmp" && \
	git init && \
	git remote add origin "https://github.com/cosmos/cosmos-proto.git" && \
	git config core.sparseCheckout true && \
	printf "proto\n" > .git/info/sparse-checkout && \
	git pull origin "$DEPS_COSMOS_PROTO" && \
	rm -f ./proto/buf.* && \
	mv ./proto/* ..
rm -rf "$THIRD_PARTY_DIR/cosmos_proto_tmp"

mkdir -p "$THIRD_PARTY_DIR/gogoproto" && \
	curl -SSL "https://raw.githubusercontent.com/cosmos/gogoproto/$DEPS_COSMOS_GOGOPROTO/gogoproto/gogo.proto" > "$THIRD_PARTY_DIR/gogoproto/gogo.proto"

mkdir -p "$THIRD_PARTY_DIR/google/api" && \
	curl -sSL https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto > "$THIRD_PARTY_DIR/google/api/annotations.proto"
	curl -sSL https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto > "$THIRD_PARTY_DIR/google/api/http.proto"

mkdir -p "$THIRD_PARTY_DIR/cosmos/ics23/v1" && \
	curl -sSL "https://raw.githubusercontent.com/cosmos/ics23/$DEPS_COSMOS_ICS23/proto/cosmos/ics23/v1/proofs.proto" > "$THIRD_PARTY_DIR/cosmos/ics23/v1/proofs.proto"

cd ${THIRD_PARTY_DIR}/../..

cat proto/buf.yaml | yq '.deps | map( "buf export " + . + " -o '${THIRD_PARTY_DIR}'") | join(" && ")' | xargs bash -c

