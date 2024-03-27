package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type BaseKeeper struct {
	bankkeeper.BaseKeeper

	ak                banktypes.AccountKeeper
	dk                distrkeeper.Keeper
	distrStoreService store.KVStoreService
	cdc               codec.BinaryCodec
}

func NewBaseKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	distrStoreService store.KVStoreService,
	ak banktypes.AccountKeeper,
	dk distrkeeper.Keeper,
	blockedAddrs map[string]bool,
	authority string,
	logger log.Logger,
) BaseKeeper {
	return BaseKeeper{
		BaseKeeper:        bankkeeper.NewBaseKeeper(cdc, storeService, ak, blockedAddrs, authority, logger),
		ak:                ak,
		dk:                dk,
		distrStoreService: distrStoreService,
		cdc:               cdc,
	}
}

func (k BaseKeeper) BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error {
	switch moduleName {
	case govtypes.ModuleName, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName:
		// send coins to distribution module
		if err := k.SendCoinsFromModuleToModule(ctx, moduleName, distrtypes.ModuleName, amounts); err != nil {
			return err
		}

		// update fee pool
		sb := collections.NewSchemaBuilder(k.distrStoreService)
		feePoolItem := collections.NewItem(sb, distrtypes.FeePoolKey, "fee_pool", codec.CollValue[distrtypes.FeePool](k.cdc))

		feePool, err := feePoolItem.Get(ctx)
		if err != nil {
			return err
		}

		coins := sdk.NewDecCoinsFromCoins(amounts...)
		feePool.CommunityPool = feePool.CommunityPool.Add(coins...)
		if err := feePoolItem.Set(ctx, feePool); err != nil {
			return err
		}

		return nil
	}

	return k.BaseKeeper.BurnCoins(ctx, moduleName, amounts)
}
