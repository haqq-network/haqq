set -euxo pipefail
echo "Bootstrapping"

HAQQD_DIR="$HOME"/.haqqd
BIN_DIR="$HAQQD_DIR"/cosmovisor/genesis/bin

if [ -f "$HAQQD_DIR"/.bootstrapped ]; then
    echo "haqqd already bootstrapped"
    exit 0
fi
echo "Bootstrapping ~/.haqqd"
# cp -r ${haqqdBinary}/share/haqqd/init "$HAQQD_DIR"
# chmod -R 0770 "$HAQQD_DIR"
id

mkdir -p "$BIN_DIR"

tmpDir=$(mktemp -d -p /tmp)
cd "$tmpDir"

haqqd_path=$(realpath "$(which haqqd)")
echo "$haqqd_path"

cp "$haqqd_path" "$BIN_DIR"
chmod -R 0771 "$BIN_DIR"/haqqd

export PATH="$BIN_DIR":$PATH
haqqd config chain-id haqq_11235-1
haqqd init "haqq-node" --chain-id haqq_11235-1

curl -L https://raw.githubusercontent.com/haqq-network/mainnet/master/genesis.json -o "$HAQQD_DIR"/config/genesis.json
# curl -L https://raw.githubusercontent.com/haqq-network/mainnet/master/addrbook.json -o "$HAQQD_DIR"/config/addrbook.json

touch "$HAQQD_DIR"/.bootstrapped
