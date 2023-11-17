#!/usr/bin/env bash
set -euxo pipefail

SWAGGER_PROTO_DIR=swagger-proto
SWAGGER_TMP_DIR=tmp-swagger-gen

mkdir -p "${SWAGGER_PROTO_DIR}/proto"

printf "version: v1\ndirectories:\n  - proto\n  - third_party" > "${SWAGGER_PROTO_DIR}/buf.work.yaml"
printf "version: v1\nname: buf.build/haqq-network/haqq\n" > "$SWAGGER_PROTO_DIR/proto/buf.yaml"

ln -snf ../../proto/buf.gen.swagger.yaml ${SWAGGER_PROTO_DIR}/proto/buf.gen.swagger.yaml

ln -snf ../../proto/haqq "${SWAGGER_PROTO_DIR}/proto/haqq"
ln -snf ../../proto/evmos "${SWAGGER_PROTO_DIR}/proto/evmos"
ln -snf ../../proto/ethermint "${SWAGGER_PROTO_DIR}/proto/ethermint"

# intermediate results of buf generate
mkdir -p $SWAGGER_TMP_DIR

cd $SWAGGER_PROTO_DIR

PATHS=""
proto_dirs=$(find -L ./proto ./third_party -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
    
    # generate swagger files (filter query files)
    query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
    if [[ -n "$query_file" ]]; then
	PATHS="$PATHS --path $query_file"
    fi
done

eval "buf generate --template proto/buf.gen.swagger.yaml $PATHS"

cd ..

cat tmp-swagger-gen/apidocs.swagger.json | jq '.info.title |= "Haqq gRPC Gateway API"' | jq '.info.version |= "0.1.0"' > client/docs/swagger-ui/swagger.json
# cp tmp-swagger-gen/apidocs.swagger.yaml client/docs/swagger-ui/swagger.yaml

# generate binary for static server
statik -src=./client/docs/swagger-ui -dest=./client/docs -f

# rm -rf $SWAGGER_TMP_DIR $SWAGGER_PROTO_DIR
