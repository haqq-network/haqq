package v162

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.6.2
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	dk distrkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("run migration v1.6.2")

		if err := fixEvergreen(ctx, bk, dk); err != nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "failed to update evergreen data: %s", err)
		}

		if err := fixVestingAccounts(ctx, ak); err != nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "failed to fix vesting accounts: %s", err)
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
