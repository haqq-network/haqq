package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type BaseKeeper struct {
	bankkeeper.BaseKeeper

	ak            banktypes.AccountKeeper
	dk            distrkeeper.Keeper
	distrStoreKey sdk.StoreKey
	cdc           codec.BinaryCodec
}

func NewBaseKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	distrStoreKey sdk.StoreKey,
	ak banktypes.AccountKeeper,
	dk distrkeeper.Keeper,
	paramSpace paramtypes.Subspace,
	blockedAddrs map[string]bool,
) BaseKeeper {
	return BaseKeeper{
		BaseKeeper:    bankkeeper.NewBaseKeeper(cdc, storeKey, ak, paramSpace, blockedAddrs),
		ak:            ak,
		dk:            dk,
		distrStoreKey: distrStoreKey,
		cdc:           cdc,
	}
}

func (k BaseKeeper) BurnCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	switch moduleName {
	case govtypes.ModuleName, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName:
		// send coins to distribution module
		if err := k.SendCoinsFromModuleToModule(ctx, moduleName, distrtypes.ModuleName, amounts); err != nil {
			return err
		}

		// update fee pool
		kvstore := ctx.MultiStore().GetKVStore(k.distrStoreKey)
		feePoolBin := kvstore.Get(distrtypes.FeePoolKey)

		var feePool distrtypes.FeePool
		k.cdc.MustUnmarshal(feePoolBin, &feePool)

		coins := sdk.NewDecCoinsFromCoins(amounts...)
		feePool.CommunityPool = feePool.CommunityPool.Add(coins...)

		b := k.cdc.MustMarshal(&feePool)
		kvstore.Set(distrtypes.FeePoolKey, b)

		return nil
	}

	return k.BaseKeeper.BurnCoins(ctx, moduleName, amounts)
}
