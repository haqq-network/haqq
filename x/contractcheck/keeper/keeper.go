package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/haqq-network/haqq/x/contractcheck/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	evmKeeper types.EVMKeeper
}

// NewKeeper creates new Keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ek types.EVMKeeper,
) Keeper {

	return Keeper{
		cdc:       cdc,
		storeKey:  storeKey,
		evmKeeper: ek,
	}
}
