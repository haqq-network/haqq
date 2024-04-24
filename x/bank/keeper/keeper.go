package keeper

import (
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

type BaseKeeper struct {
	bankkeeper.BaseKeeper

	ak  AccountKeeper
	ek  ERC20Keeper
	dk  DistributionKeeper
	cdc codec.BinaryCodec
}

func NewBaseKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	ak AccountKeeper,
	ek ERC20Keeper,
	dk DistributionKeeper,
	blockedAddrs map[string]bool,
	authority string,
	logger log.Logger,
) BaseKeeper {
	return BaseKeeper{
		BaseKeeper: bankkeeper.NewBaseKeeper(cdc, storeService, ak, blockedAddrs, authority, logger),
		ak:         ak,
		ek:         ek,
		dk:         dk,
		cdc:        cdc,
	}
}
