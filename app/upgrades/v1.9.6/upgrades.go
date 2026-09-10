package v196

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"

	stakingkeeper "github.com/haqq-network/haqq/x/staking/keeper"
	ucdaokeeper "github.com/haqq-network/haqq/x/ucdao/keeper"
)

// Keepers groups the keepers the v1.9.6 handler touches.
type Keepers struct {
	AccountKeeper  authkeeper.AccountKeeper
	BankKeeper     bankkeeper.Keeper
	StakingKeeper  stakingkeeper.Keeper
	DistrKeeper    distrkeeper.Keeper
	AuthzKeeper    authzkeeper.Keeper
	FeeGrantKeeper feegrantkeeper.Keeper
	DaoKeeper      ucdaokeeper.Keeper
}

// CreateUpgradeHandler creates an SDK upgrade handler for Haqq v1.9.6
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers Keepers,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		logger := ctx.Logger().With("upgrade", UpgradeName)

		// run the v1.9.6 migrations
		logger.Info("Running module migrations...")
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		// Any error here fails the upgrade on every node at the same height:
		// a partial burn must never be committed.
		logger.Info("Burning funds of frozen accounts...")
		if err := BurnHackerFunds(ctx, keepers, HackerAccounts); err != nil {
			return vm, errorsmod.Wrap(err, "failed to burn funds of frozen accounts")
		}

		return vm, nil
	}
}
