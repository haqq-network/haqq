#!/usr/bin/env bash
set -euxo pipefail

# Save the original directory
ORIGINAL_DIR="$(pwd)"

# Check if THIRD_PARTY_DIR is set
if [ -z "${THIRD_PARTY_DIR:-}" ]; then
	echo "Error: THIRD_PARTY_DIR is not set. This script should be run via 'make proto-swagger-download-deps'"
	exit 1
fi

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

# Return to the original directory
cd "${ORIGINAL_DIR}"

# Export buf dependencies without requiring yq
# Read deps from buf.yaml and export each one
if ! command -v yq &> /dev/null; then
	# Fallback: manually extract deps from buf.yaml and export them
	# This works for the current buf.yaml structure
	buf export buf.build/cosmos/cosmos-sdk -o ${THIRD_PARTY_DIR}
	buf export buf.build/cosmos/cosmos-proto -o ${THIRD_PARTY_DIR}
	buf export buf.build/cosmos/gogo-proto -o ${THIRD_PARTY_DIR}
	buf export buf.build/googleapis/googleapis -o ${THIRD_PARTY_DIR}
else
	# Use yq if available (preferred method)
	cat proto/buf.yaml | yq '.deps | map( "buf export " + . + " -o '${THIRD_PARTY_DIR}'") | join(" && ")' | xargs bash -c
fi

