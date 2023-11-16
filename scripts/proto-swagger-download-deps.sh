#!/usr/bin/env bash
set -euxo pipefail

mkdir -p $THIRD_PARTY_DIR
cat proto/buf.yaml | yq '.deps | map( "buf export " + . + " -o '${THIRD_PARTY_DIR}'") | join(" && ")' | xargs bash -c

