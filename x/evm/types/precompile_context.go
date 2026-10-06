package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// PrecompileStateDB is the part of the calling transaction's StateDB that code
// reached from a stateful precompile writes EVM account state through.
type PrecompileStateDB interface {
	GetCodeHash(addr common.Address) common.Hash
	SetCode(addr common.Address, code []byte)
}

// precompileContextKey marks a context handed to a stateful precompile.
type precompileContextKey struct{}

// precompileFrame is the value stored under precompileContextKey.
type precompileFrame struct {
	stateDB PrecompileStateDB
}

// WithPrecompileContext marks ctx as the context a stateful precompile runs in
// and attaches the StateDB of the transaction that called it.
//
// A precompile works on that StateDB's cache context while the StateDB still
// holds the transaction's EVM state in memory. Anything written to EVM state in
// the cache context behind its back - storage by a nested EVM call, an account
// or its code through the x/evm keeper - is invisible to it: it keeps serving,
// and finally re-commits, the values it already had. Code running on a marked
// context therefore may not write EVM state through the keeper or commit a
// nested EVM call; it has to go through the attached StateDB, whose journal
// also reverts the write together with the precompile call.
func WithPrecompileContext(ctx sdk.Context, stateDB PrecompileStateDB) sdk.Context {
	return ctx.WithValue(precompileContextKey{}, precompileFrame{stateDB: stateDB})
}

// IsPrecompileContext reports whether ctx, or a context derived from it, was
// handed to a stateful precompile.
func IsPrecompileContext(ctx sdk.Context) bool {
	_, marked := ctx.Value(precompileContextKey{}).(precompileFrame)
	return marked
}

// PrecompileStateDBFromContext returns the StateDB of the transaction whose
// stateful precompile ctx was handed to, if any.
func PrecompileStateDBFromContext(ctx sdk.Context) (PrecompileStateDB, bool) {
	frame, marked := ctx.Value(precompileContextKey{}).(precompileFrame)
	if !marked || frame.stateDB == nil {
		return nil, false
	}
	return frame.stateDB, true
}
