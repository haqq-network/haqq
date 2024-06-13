package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
)

// GetParams returns the total set of liquidvesting parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the liquidvesting parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	k.paramstore.SetParamSet(ctx, &params)

	return nil
}

// GetCACContractAddress gets the CAC contract address
func (k Keeper) GetCACContractAddress(ctx sdk.Context) string {
	params := k.GetParams(ctx)
	return params.CacContractAddress
}

// SetCACContractAddress sets the CAC contract address
func (k Keeper) SetCACContractAddress(ctx sdk.Context, address string) {
	params := k.GetParams(ctx)
	params.CacContractAddress = address
	err := k.SetParams(ctx, params)
	if err != nil {
		panic(err)
	}
}

// ResetParamsToDefault resets the params to the default values
func (k Keeper) ResetParamsToDefault(ctx sdk.Context) {
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	if err != nil {
		panic(err)
	}
}
