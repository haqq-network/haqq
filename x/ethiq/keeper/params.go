package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

// IsModuleEnabled checks if module enabled in params
func (k Keeper) IsModuleEnabled(ctx sdk.Context) bool {
	params := k.GetParams(ctx)
	return params.Enabled
}

// GetParams returns the total set of ethiq parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the ethiq parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}
