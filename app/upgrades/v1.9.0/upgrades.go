package v190

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ethereum/go-ethereum/common"

	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for Evmos v20
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	gk govkeeper.Keeper,
	ek erc20keeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		logger := ctx.Logger().With("upgrade", UpgradeName)

		// run the sdk v0.50 migrations
		logger.Debug("Running module migrations...")
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		logger.Debug("Updating expedited prop params...")
		if err := UpdateExpeditedPropsParams(ctx, gk); err != nil {
			logger.Error("error while updating gov params", "error", err.Error())
			return nil, err
		}

		logger.Debug("Register dynamic precompiles...")
		if err := RegisterDynamicPrecompiles(ctx, ek); err != nil {
			logger.Error("error while registering erc20 precompiles", "error", err.Error())
			return nil, err
		}

		return vm, nil
	}
}

func UpdateExpeditedPropsParams(ctx sdk.Context, gk govkeeper.Keeper) error {
	params, err := gk.Params.Get(ctx)
	if err != nil {
		return err
	}

	// use the same denom as the min deposit denom
	// also amount must be greater than MinDeposit amount
	denom := params.MinDeposit[0].Denom
	expDepAmt := params.ExpeditedMinDeposit[0].Amount
	if expDepAmt.LTE(params.MinDeposit[0].Amount) {
		expDepAmt = params.MinDeposit[0].Amount.MulRaw(govv1.DefaultMinExpeditedDepositTokensRatio)
	}
	params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(denom, expDepAmt))

	// if expedited voting period > voting period
	// set expedited voting period to be half the voting period
	if params.ExpeditedVotingPeriod != nil && params.VotingPeriod != nil && *params.ExpeditedVotingPeriod > *params.VotingPeriod {
		expPeriod := *params.VotingPeriod / 2
		params.ExpeditedVotingPeriod = &expPeriod
	}

	if err := params.ValidateBasic(); err != nil {
		return err
	}
	return gk.Params.Set(ctx, params)
}

func RegisterDynamicPrecompiles(ctx sdk.Context, ek erc20keeper.Keeper) error {
	erc20Params := ek.GetParams(ctx)
	if len(erc20Params.DynamicPrecompiles) == 0 {
		return nil
	}

	// if a precompile is present we should register the account with the erc20 codehash
	for _, precompile := range erc20Params.DynamicPrecompiles {
		if err := ek.RegisterERC20CodeHash(ctx, common.HexToAddress(precompile)); err != nil {
			return err
		}
	}

	return nil
}
