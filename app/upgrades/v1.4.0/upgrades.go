package v140

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/haqq-network/haqq/types"
	coinomicskeeper "github.com/haqq-network/haqq/x/coinomics/keeper"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"

	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"

	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.4.0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	sk stakingkeeper.Keeper,
	ck coinomicskeeper.Keeper,
	slashk slashingkeeper.Keeper,
	gk govkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("### Run migration v1.4.0 ###")

		UpdateStakingParams(ctx, sk)
		UpdateSlashingParams(ctx, slashk)
		UpdateGovParams(ctx, gk, sk)

		// Reset coinomics state for TestEdge2
		if types.IsTestEdge2Network(ctx.ChainID()) {
			if err := ResetCoinomicsState(ctx, sk, ck); err != nil {
				logger.Error("FAILED: reset coinomics params error: ", err.Error())
			}
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}

func UpdateGovParams(ctx sdk.Context, gk govkeeper.Keeper, sk stakingkeeper.Keeper) {
	depositParams := gk.GetParams(ctx)

	if types.IsMainNetwork(ctx.ChainID()) {
		minDeposit := math.NewIntWithDecimal(6_000, 18) // 6 000 ISLM
		depositParams.MinDeposit = sdk.NewCoins(
			sdk.NewCoin(sk.BondDenom(ctx), minDeposit),
		)
	} else {
		minDeposit := math.NewIntWithDecimal(60_000, 18) // 60 000 ISLM
		depositParams.MinDeposit = sdk.NewCoins(
			sdk.NewCoin(sk.BondDenom(ctx), minDeposit),
		)
	}

	if err := gk.SetParams(ctx, depositParams); err != nil {
		panic(err)
	}
}

func UpdateSlashingParams(ctx sdk.Context, slashingkeeper slashingkeeper.Keeper) {
	params := slashingkeeper.GetParams(ctx)
	params.SignedBlocksWindow = 35000
	params.SlashFractionDowntime = sdk.NewDecWithPrec(1, 4) // 0.01% (0.0001)
	if err := slashingkeeper.SetParams(ctx, params); err != nil {
		panic(err)
	}
}

func UpdateStakingParams(ctx sdk.Context, sk stakingkeeper.Keeper) {
	params := sk.GetParams(ctx)
	params.MaxValidators = 150
	params.MinCommissionRate = sdk.NewDecWithPrec(5, 2) // 5% (0.05)
	if err := sk.SetParams(ctx, params); err != nil {
		panic(err)
	}
}

func ResetCoinomicsState(ctx sdk.Context, sk stakingkeeper.Keeper, ck coinomicskeeper.Keeper) error {
	newCoinomicsParams := coinomicstypes.DefaultParams()
	newCoinomicsParams.MintDenom = sk.BondDenom(ctx)
	maxSupply := ck.GetMaxSupply(ctx)
	doubledMaxSupply := maxSupply.Amount.Mul(sdk.NewInt(2))

	ck.SetParams(ctx, newCoinomicsParams)
	ck.SetEra(ctx, 0)
	ck.SetMaxSupply(ctx, sdk.NewCoin(newCoinomicsParams.MintDenom, doubledMaxSupply))

	return nil
}
