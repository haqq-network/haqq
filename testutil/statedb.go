package testutil

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/app/ante/evm"
	"github.com/haqq-network/haqq/x/evm/statedb"
)

// NewStateDB returns a new StateDB for testing purposes.
func NewStateDB(ctx sdk.Context, evmKeeper evm.EVMKeeper) *statedb.StateDB {
	return statedb.New(ctx, evmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash().Bytes())))
}
