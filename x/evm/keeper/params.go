package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
)

// GetParams returns the total set of evm parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params evmtypes.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(evmtypes.KeyPrefixParams)
	if len(bz) == 0 {
		return k.GetLegacyParams(ctx)
	}
	k.cdc.MustUnmarshal(bz, &params)
	return
}

// SetParams sets the EVM params each in their individual key for better get performance
func (k Keeper) SetParams(ctx sdk.Context, params evmtypes.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(evmtypes.KeyPrefixParams, bz)
	return nil
}

// GetLegacyParams returns param set for version before migrate
func (k Keeper) GetLegacyParams(ctx sdk.Context) evmtypes.Params {
	var params evmtypes.Params
	k.ss.GetParamSetIfExists(ctx, &params)
	return params
}
