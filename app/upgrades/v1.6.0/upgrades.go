package v160

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	genesistypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/genesis/types"
	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"

	ica "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts"
	icahosttypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"

	errorsmod "cosmossdk.io/errors"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.6.0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("run migration v1.6.0")

		if err := restoreAccounts(ctx, ak); err != nil {
			panic(err)
		}

		// create ICS27 Controller submodule params, with the controller module NOT enabled
		gs := &genesistypes.GenesisState{
			ControllerGenesisState: genesistypes.ControllerGenesisState{},
			HostGenesisState: genesistypes.HostGenesisState{
				Port: icatypes.HostPortID,
				Params: icahosttypes.Params{
					HostEnabled: true,
					AllowMessages: []string{
						sdk.MsgTypeURL(&banktypes.MsgSend{}),
						sdk.MsgTypeURL(&banktypes.MsgMultiSend{}),
						sdk.MsgTypeURL(&distrtypes.MsgSetWithdrawAddress{}),
						sdk.MsgTypeURL(&distrtypes.MsgWithdrawDelegatorReward{}),
						sdk.MsgTypeURL(&govtypes.MsgVote{}),
						sdk.MsgTypeURL(&govtypes.MsgVoteWeighted{}),
						sdk.MsgTypeURL(&stakingtypes.MsgDelegate{}),
						sdk.MsgTypeURL(&stakingtypes.MsgUndelegate{}),
						sdk.MsgTypeURL(&stakingtypes.MsgCancelUnbondingDelegation{}),
						sdk.MsgTypeURL(&stakingtypes.MsgBeginRedelegate{}),
						sdk.MsgTypeURL(&transfertypes.MsgTransfer{}),
					},
				},
			},
		}

		bz, err := icatypes.ModuleCdc.MarshalJSON(gs)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed to marshal %s genesis state", icatypes.ModuleName)
		}

		// Register the consensus version in the version map to avoid the SDK from triggering the default
		// InitGenesis function.
		vm[icatypes.ModuleName] = ica.AppModule{}.ConsensusVersion()

		_ = mm.Modules[icatypes.ModuleName].InitGenesis(ctx, icatypes.ModuleCdc, bz)

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
