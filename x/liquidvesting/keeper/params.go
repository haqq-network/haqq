package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

// var isTrue = []byte("0x01")

// GetParams returns the total set of liquidvesting parameters.
func (k BaseKeeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the liquidvesting parameters to the param space.
func (k BaseKeeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	k.paramstore.SetParamSet(ctx, &params)

	return nil
}

func (k BaseKeeper) IsLiquidVestingEnabled(ctx sdk.Context) bool {
	params := k.GetParams(ctx)
	return params.EnableLiquidVesting
}

func (k BaseKeeper) SetLiquidVestingEnabled(ctx sdk.Context, enable bool) {
	params := k.GetParams(ctx)
	params.EnableLiquidVesting = enable
	err := k.SetParams(ctx, params)
	if err != nil {
		panic(err)
	}
}

func (k BaseKeeper) ResetParamsToDefault(ctx sdk.Context) {
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	if err != nil {
		panic(err)
	}
}
