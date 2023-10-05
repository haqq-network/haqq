package v160

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctmmigrations "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint/migrations"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"

	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	genesistypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/genesis/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	errorsmod "cosmossdk.io/errors"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.6.0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ck consensuskeeper.Keeper,
	clientKeeper ibctmmigrations.ClientKeeper,
	pk paramskeeper.Keeper,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("run migration v1.6.0")

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
						sdk.MsgTypeURL(&govtypesv1.MsgVote{}),
						sdk.MsgTypeURL(&govtypesv1.MsgVoteWeighted{}),
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

		m := mm.Modules[icatypes.ModuleName].(module.HasGenesis)
		m.InitGenesis(ctx, icatypes.ModuleCdc, bz)

		// ----------------------------------------

		// Set param key table for params module migration
		for _, subspace := range pk.GetSubspaces() {
			var keyTable paramstypes.KeyTable
			switch subspace.Name() {
			case authtypes.ModuleName:
				keyTable = authtypes.ParamKeyTable() //nolint:staticcheck
			case banktypes.ModuleName:
				keyTable = banktypes.ParamKeyTable() //nolint:staticcheck,nolintlint
			case stakingtypes.ModuleName:
				keyTable = stakingtypes.ParamKeyTable()
			case minttypes.ModuleName:
				keyTable = minttypes.ParamKeyTable() //nolint:staticcheck
			case distrtypes.ModuleName:
				keyTable = distrtypes.ParamKeyTable() //nolint:staticcheck,nolintlint
			case slashingtypes.ModuleName:
				keyTable = slashingtypes.ParamKeyTable() //nolint:staticcheck
			case govtypes.ModuleName:
				keyTable = govv1.ParamKeyTable() //nolint:staticcheck
			case crisistypes.ModuleName:
				keyTable = crisistypes.ParamKeyTable() //nolint:staticcheck
			case ibctransfertypes.ModuleName:
				keyTable = ibctransfertypes.ParamKeyTable()
			case evmtypes.ModuleName:
				keyTable = evmtypes.ParamKeyTable() //nolint:staticcheck
			case feemarkettypes.ModuleName:
				keyTable = feemarkettypes.ParamKeyTable()
			default:
				continue
			}
			if !subspace.HasKeyTable() {
				subspace.WithKeyTable(keyTable)
			}
		}

		baseAppLegacySS := pk.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())

		baseapp.MigrateParams(ctx, baseAppLegacySS, &ck)

		if _, err := ibctmmigrations.PruneExpiredConsensusStates(ctx, cdc, clientKeeper); err != nil {
			return nil, err
		}

		//

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
