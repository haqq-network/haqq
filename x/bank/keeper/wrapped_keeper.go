package keeper

import (
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

type WrappedBaseKeeper struct {
	bankkeeper.Keeper
	ek ERC20Keeper
	ak AccountKeeper
}

func NewWrappedBaseKeeper(
	bk bankkeeper.Keeper,
	ek ERC20Keeper,
	ak AccountKeeper,
) WrappedBaseKeeper {
	return WrappedBaseKeeper{
		Keeper: bk,
		ek:     ek,
		ak:     ak,
	}
}

func (k WrappedBaseKeeper) UnwrapKeeper() bankkeeper.Keeper {
	return k.Keeper
}

func (k WrappedBaseKeeper) UnwrapBaseKeeper() bankkeeper.BaseKeeper {
	return k.Keeper.(bankkeeper.BaseKeeper)
}
