#!/usr/bin/env bash
set -euxo pipefail

# Save the original directory
ORIGINAL_DIR="$(pwd)"

# Check if THIRD_PARTY_DIR is set
if [ -z "${THIRD_PARTY_DIR:-}" ]; then
	echo "Error: THIRD_PARTY_DIR is not set. This script should be run via 'make proto-swagger-download-deps'"
	exit 1
fi

# Extract IBC version - match specifically github.com/cosmos/ibc-go/v* (main IBC package)
IBC_VERSION=$(grep "^[[:space:]]*github.com/cosmos/ibc-go/v[0-9]" go.mod | awk '{print $2}' | head -1)
# Extract ICS23 version - match github.com/cosmos/ics23/go
ICS_VERSION=$(grep "^[[:space:]]*github.com/cosmos/ics23/go" go.mod | awk '{print $2}' | head -1)

if [ -z "${IBC_VERSION}" ]; then
	echo "Error: Could not find github.com/cosmos/ibc-go/v* in go.mod"
	exit 1
fi

mkdir -p $THIRD_PARTY_DIR

# reuse buf.yaml after upgrading ibc

# Clean up any existing ibc_tmp directory first
rm -rf "${THIRD_PARTY_DIR}/ibc_tmp"

# Download IBC proto files via git
mkdir -p "${THIRD_PARTY_DIR}/ibc_tmp" && \
	cd "${THIRD_PARTY_DIR}/ibc_tmp" && \
	git init && \
	git remote add origin "https://github.com/cosmos/ibc-go.git" && \
	git config core.sparseCheckout true && \
	printf "proto\n" > .git/info/sparse-checkout && \
	git pull origin ${IBC_VERSION} --depth 1 && \
	rm -f ./proto/buf.* && \
	mv ./proto/* "${THIRD_PARTY_DIR}/" && \
	cd "${THIRD_PARTY_DIR}" && \
	rm -rf "${THIRD_PARTY_DIR}/ibc_tmp"

# Return to the original directory
cd "${ORIGINAL_DIR}"

# Export buf dependencies - ICS23 proto files need to be in cosmos/ics23/v1/ structure
# ICS23 proto files are available via buf.build and must be exported
if ! command -v yq &> /dev/null; then
	# Fallback: manually extract deps from buf.yaml and export them
	# This works for the current buf.yaml structure
	buf export buf.build/cosmos/cosmos-sdk -o ${THIRD_PARTY_DIR}
	buf export buf.build/cosmos/cosmos-proto -o ${THIRD_PARTY_DIR}
	buf export buf.build/cosmos/gogo-proto -o ${THIRD_PARTY_DIR}
	buf export buf.build/googleapis/googleapis -o ${THIRD_PARTY_DIR}
	buf export buf.build/cosmos/ics23 -o ${THIRD_PARTY_DIR}
else
	# Use yq if available (preferred method)
	# Iterate over each dependency and export it
	yq -r '.deps[]' proto/buf.yaml | while read -r dep; do
		buf export "$dep" -o "${THIRD_PARTY_DIR}"
	done
	# Ensure ICS23 is exported even if not in buf.yaml deps
	buf export buf.build/cosmos/ics23 -o ${THIRD_PARTY_DIR}
fi

