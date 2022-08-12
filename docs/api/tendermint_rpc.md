# Tendermint RPC


## Tendermint supports the following RPC protocols:

URI over HTTP

### Configuration

RPC can be configured by tuning parameters under [rpc] table in the ```$TMHOME/config/config.toml``` file or by using the ```--rpc.X``` command-line flags.

Default rpc listen address is ```tcp://0.0.0.0:26657```. To set another address, set the laddr config parameter to desired value. CORS (Cross-Origin Resource Sharing) can be enabled by setting ```cors_allowed_origins```, ```cors_allowed_methods```, ```cors_allowed_headers config parameters```.

### Arguments

Arguments which expect strings or byte arrays may be passed as quoted strings, like "abc" or as 0x-prefixed strings, like ```0x616263```.

### URI/HTTP
A REST like interface.

```sh
curl localhost:26657/block?height=5
```

### JSON-RPC/HTTP
JSONRPC requests can be POST'd to the root RPC endpoint via HTTP.

```sh
curl --header "Content-Type: application/json" --request POST --data '{"method": "block", "params": ["5"], "id": 1}' localhost:26657
```
### JSON-RPC/websockets

JSONRPC requests can be also made via websocket. The websocket endpoint is at /websocket, e.g. ```localhost:26657/websocket```. Asynchronous RPC functions like event subscribe and unsubscribe are only available via websockets.

Example using [ws](https://github.com/hashrocket/ws):

```sh
ws ws://localhost:26657/websocket
> { "jsonrpc": "2.0", "method": "subscribe", "params": ["tm.event='NewBlock'"], "id": 1 }
```

## gRPC & REST endpoints

**Haqq Mainnet**:

- [RPC - https://rpc.tm.haqq.network](https://rpc.tm.haqq.network)

**Haqq TestEdge**:

- [RPC - https://rpc.tm.testedge.haqq.network](https://rpc.tm.testedge.haqq.network)

## Tendermint Swagger

Swagger: [Tendermint RPC](https://docs.tendermint.com/v0.34/rpc/)
