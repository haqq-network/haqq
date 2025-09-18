package keeper

import (
	"context"

	"cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Keeper is a wrapper around the Cosmos SDK bank keeper.
type Keeper struct {
	bankkeeper.BaseKeeper

	ak                     banktypes.AccountKeeper
	dk                     distrkeeper.Keeper
	cdc                    codec.BinaryCodec
	storeService           store.KVStoreService
	mintCoinsRestrictionFn banktypes.MintingRestrictionFn
	logger                 log.Logger
}

// NewKeeper creates a new staking Keeper wrapper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	ak banktypes.AccountKeeper,
	dk distrkeeper.Keeper,
	blockedAddrs map[string]bool,
	authority string,
	logger log.Logger,
) Keeper {
	return Keeper{
		BaseKeeper: bankkeeper.NewBaseKeeper(cdc, storeService, ak, blockedAddrs, authority, logger),
		ak:         ak,
		dk:         dk,
		cdc:        cdc,
	}
}

func (k Keeper) BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error {
	switch moduleName {
	case govtypes.ModuleName, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName:
		acc := k.ak.GetModuleAccount(ctx, moduleName)
		if acc == nil {
			panic(errorsmod.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", moduleName))
		}

		return k.dk.FundCommunityPool(ctx, amounts, acc.GetAddress())
	}

	return k.BaseKeeper.BurnCoins(ctx, moduleName, amounts)
}
