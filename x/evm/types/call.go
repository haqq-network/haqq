// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package types

type CallType int

const (
	// RPC call type is used on requests to eth_estimateGas rpc API endpoint
	RPC CallType = iota + 1
	// Internal call type is used in case of smart contract methods calls
	Internal
)

// MaxPrecompileCalls is the maximum number of stateful precompile
// calls within a transaction. We want to limit this because
// each call snapshots the cached context and flushes the dirty
// StateDB into it, so every call that gets past calldata
// validation counts, whether it then succeeds or fails.
const MaxPrecompileCalls uint8 = 7
