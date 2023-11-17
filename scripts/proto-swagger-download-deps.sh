#!/usr/bin/env bash
set -euxo pipefail

IBC_VERSION=$(grep "github.com/cosmos/ibc" go.mod | awk '{print $2}')
ICS_VERSION=$(grep "github.com/confio/ics23" go.mod | awk '{print $2}')

mkdir -p $THIRD_PARTY_DIR

# reuse buf.yaml after upgrading ibc

mkdir -p "${THIRD_PARTY_DIR}/ibc_tmp" && \
	cd "${THIRD_PARTY_DIR}/ibc_tmp" && \
	git init && \
	git remote add origin "https://github.com/cosmos/ibc-go.git" && \
	git config core.sparseCheckout true && \
	printf "proto\n" > .git/info/sparse-checkout && \
	git pull origin ${IBC_VERSION} --depth 1 && \
	rm -f ./proto/buf.* && \
	mv ./proto/* .. && \
	rm -rf "${THIRD_PARTY_DIR}/ibc_tmp"

mkdir -p "${THIRD_PARTY_DIR}/ics_tmp" && \
	cd "${THIRD_PARTY_DIR}/ics_tmp" && \
	git init && \
	git remote add origin "https://github.com/confio/ics23.git" && \
	git fetch origin --tags && \
	git checkout go/${ICS_VERSION} && \
	mv ./*.proto .. && \
	rm -rf "${THIRD_PARTY_DIR}/ics_tmp"

cd ${THIRD_PARTY_DIR}/../..

cat proto/buf.yaml | yq '.deps | map( "buf export " + . + " -o '${THIRD_PARTY_DIR}'") | join(" && ")' | xargs bash -c

