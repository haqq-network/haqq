package v161

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/haqq-network/haqq/utils"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.6.1
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("run migration v1.6.1")

		if utils.IsMainNetwork(ctx.ChainID()) {
			if err := restoreAccounts(ctx, ak); err != nil {
				return nil, errorsmod.Wrap(err, "failed to restore accounts data")
			}
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}

func restoreAccounts(ctx sdk.Context, ak authkeeper.AccountKeeper) error {
	logger := ctx.Logger()
	logger.Info("Restoring accounts data...")

	for _, accData := range accounts {
		accAddr := sdk.MustAccAddressFromBech32(accData.addr)
		acc := ak.GetAccount(ctx, accAddr)
		if acc == nil {
			return fmt.Errorf("account not found: %s", accData.addr)
		}

		oldNum := acc.GetAccountNumber()
		if err := acc.SetAccountNumber(accData.accNum); err != nil {
			return fmt.Errorf("failed to restore account number: %s: %d", accData.addr, accData.accNum)
		}

		oldSeq := acc.GetSequence()
		newSeq := oldSeq + accData.seq
		if err := acc.SetSequence(newSeq); err != nil {
			return fmt.Errorf("failed to restore sequence: %s: %d", accData.addr, newSeq)
		}

		ak.SetAccount(ctx, acc)
		logger.Info(fmt.Sprintf("Restored account - %s: num %d -> %d; seq %d -> %d", accData.addr, oldNum, accData.accNum, oldSeq, newSeq))
	}

	return nil
}
