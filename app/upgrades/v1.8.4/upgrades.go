package v184

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v19
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	erc20k erc20keeper.Keeper,
	ek *evmkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger().With("upgrade", UpgradeName)

		// run module migrations first.
		// so we wont override erc20 params when running strv2 migration,
		migrationRes, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return migrationRes, err
		}

		// Reactivate all default static Precompiles
		ctxCache, writeFn := ctx.CacheContext()
		if err := ReactivateStaticPrecompiles(ctxCache, ek); err == nil {
			writeFn()
		} else {
			logger.Error("error removing outposts", "error", err)
		}

		// Migrate from deployed ERC20 contracts to ERC20 Precompile for all registered Cosmos native coins.
		ctxCache, writeFn = ctx.CacheContext()
		if err = RunSTRv2Migration(ctxCache, logger, ak, bk, erc20k, ek); err == nil {
			writeFn()
		} else {
			logger.Error("error running STRv2 migration", "error", err)
		}

		return migrationRes, err
	}
}

// ReactivateStaticPrecompiles sets ActiveStaticPrecompiles param on the evm
func ReactivateStaticPrecompiles(ctx sdk.Context,
	evmKeeper *evmkeeper.Keeper,
) error {
	params := evmKeeper.GetParams(ctx)
	params.ActiveStaticPrecompiles = evmtypes.DefaultStaticPrecompiles
	return evmKeeper.SetParams(ctx, params)
}

// RunSTRv2Migration converts all the registered ERC-20 tokens of Cosmos native token pairs
// back to the native representation and registers the WEVMOS token as an ERC-20 token pair.
func RunSTRv2Migration(
	ctx sdk.Context,
	logger log.Logger,
	accountKeeper authkeeper.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
	erc20Keeper erc20keeper.Keeper,
	evmKeeper *evmkeeper.Keeper,
) error {
	// Filter all token pairs for the ones that are for Cosmos native coins.
	nativeTokenPairs := getNativeTokenPairs(ctx, erc20Keeper)

	// NOTE (@fedekunze): first we must convert the all the registered tokens.
	// If we do it the other way around, the conversion will fail since there won't
	// be any contract code due to the selfdestruct.
	if err := ConvertERC20Coins(
		ctx,
		logger,
		accountKeeper,
		bankKeeper,
		*evmKeeper,
		nativeTokenPairs,
	); err != nil {
		return errorsmod.Wrap(err, "failed to convert native coins")
	}

	if err := registerERC20Extensions(ctx, erc20Keeper, evmKeeper); err != nil {
		return errorsmod.Wrap(err, "failed to register ERC-20 extensions")
	}

	return nil
}

// registerERC20Extensions registers the ERC20 precompiles with the EVM.
func registerERC20Extensions(
	ctx sdk.Context,
	erc20Keeper erc20keeper.Keeper,
	evmKeeper *evmkeeper.Keeper,
) error {
	params := erc20Keeper.GetParams(ctx)

	var err error
	erc20Keeper.IterateTokenPairs(ctx, func(tokenPair erc20types.TokenPair) bool {
		// skip registration if token is EVM native or if it has already been registered
		// NOTE: this should handle failure during the selfdestruct
		if tokenPair.ContractOwner != erc20types.OWNER_MODULE ||
			erc20Keeper.IsAvailableERC20Precompile(&params, tokenPair.GetERC20Contract()) {
			return false
		}

		address := tokenPair.GetERC20Contract()
		err = erc20Keeper.EnableDynamicPrecompiles(ctx, address)
		if err != nil {
			return true
		}

		// try selfdestruct ERC20 contract
		// NOTE(@fedekunze): From now on, the contract address will map to a precompile instead
		// of the ERC20MinterBurner contract. We try to force a selfdestruct to remove the unnecessary
		// code and storage from the state machine. In any case, the precompiles are handled in the EVM
		// before the regular contracts so not removing them doesn't create any issues in the implementation.
		err = evmKeeper.DeleteAccount(ctx, address)
		if err != nil {
			err = errorsmod.Wrapf(err, "failed to selfdestruct account %s", address)
			return true
		}

		return false
	})

	return err
}
