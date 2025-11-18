package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/gogoproto/proto"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	abci "github.com/cometbft/cometbft/abci/types"
	tmos "github.com/cometbft/cometbft/libs/os"
	dbm "github.com/cosmos/cosmos-db"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	testdata_pulsar "github.com/cosmos/cosmos-sdk/testutil/testdata/testpb"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/types/module"
	sigtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	sdkstaking "github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/ibc-go/modules/capability"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibctransfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	ica "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"

	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/types"

	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"

	"github.com/haqq-network/haqq/encoding"
	"github.com/haqq-network/haqq/x/evm/core/vm"

	// unnamed import of statik for swagger UI support
	_ "github.com/haqq-network/haqq/client/docs/statik"
	"github.com/haqq-network/haqq/utils"

	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/x/epochs"
	epochskeeper "github.com/haqq-network/haqq/x/epochs/keeper"
	epochstypes "github.com/haqq-network/haqq/x/epochs/types"
	"github.com/haqq-network/haqq/x/evm"
	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"

	"github.com/haqq-network/haqq/app/ante"
	ethante "github.com/haqq-network/haqq/app/ante/evm"
	srvflags "github.com/haqq-network/haqq/server/flags"
	"github.com/haqq-network/haqq/x/erc20"
	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/feemarket"
	feemarketkeeper "github.com/haqq-network/haqq/x/feemarket/keeper"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"
	"github.com/haqq-network/haqq/x/staking"
	stakingkeeper "github.com/haqq-network/haqq/x/staking/keeper"
	"github.com/haqq-network/haqq/x/vesting"
	vestingkeeper "github.com/haqq-network/haqq/x/vesting/keeper"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"

	haqqbank "github.com/haqq-network/haqq/x/bank"
	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
	"github.com/haqq-network/haqq/x/coinomics"
	coinomicskeeper "github.com/haqq-network/haqq/x/coinomics/keeper"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	"github.com/haqq-network/haqq/x/liquidvesting"
	liquidvestingkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	"github.com/haqq-network/haqq/x/ucdao"
	ucdaokeeper "github.com/haqq-network/haqq/x/ucdao/keeper"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"

	v1100 "github.com/haqq-network/haqq/app/upgrades/v1.10.0"
	v191 "github.com/haqq-network/haqq/app/upgrades/v1.9.1"

	// NOTE: override ICS20 keeper to support IBC transfers of ERC20 tokens
	"github.com/haqq-network/haqq/x/ibc/transfer"
	transferkeeper "github.com/haqq-network/haqq/x/ibc/transfer/keeper"

	// memiavlstore "github.com/crypto-org-chain/cronos/store"

	// Force-load the tracer engines to trigger registration due to Go-Ethereum v1.10.15 changes
	_ "github.com/haqq-network/haqq/x/evm/core/tracers/js"
	_ "github.com/haqq-network/haqq/x/evm/core/tracers/native"
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".haqqd")

	// manually update the power reduction by replacing micro (u) -> atto (a)
	sdk.DefaultPowerReduction = haqqtypes.PowerReduction
	// modify fee market parameter defaults through global
	feemarkettypes.DefaultMinGasPrice = MinGasPrices
	feemarkettypes.DefaultMinGasMultiplier = MinGasMultiplier
	// modify default min commission to 5%
	stakingtypes.DefaultMinCommissionRate = math.LegacyNewDecWithPrec(5, 2)
}

const (
	// Name defines the application binary name
	Name           = "haqqd"
	MainnetChainID = "haqq_11235"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		icatypes.ModuleName:            nil,
		evmtypes.ModuleName:            {authtypes.Minter, authtypes.Burner}, // used for secure addition and subtraction of balance using module account
		erc20types.ModuleName:          {authtypes.Minter, authtypes.Burner},
		coinomicstypes.ModuleName:      {authtypes.Minter},
		vestingtypes.ModuleName:        nil, // Add vesting module account
		liquidvestingtypes.ModuleName:  {authtypes.Minter, authtypes.Burner},
		ucdaotypes.ModuleName:          nil,
	}
)

var (
	_ servertypes.Application = (*Haqq)(nil)
	_ ibctesting.TestingApp   = (*Haqq)(nil)
	_ runtime.AppI            = (*Haqq)(nil)
)

// Haqq implements an extended ABCI application. It is an application
// that may process transactions through Ethereum's EVM running atop of
// CometBFT consensus.
type Haqq struct {
	*baseapp.BaseApp

	// encoding
	cdc               *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry
	txConfig          client.TxConfig

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	CapabilityKeeper      *capabilitykeeper.Keeper
	StakingKeeper         stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	CrisisKeeper          crisiskeeper.Keeper
	UpgradeKeeper         upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	ICAHostKeeper         icahostkeeper.Keeper
	PacketForwardKeeper   *packetforwardkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        transferkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// Ethermint keepers
	EvmKeeper       *evmkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper

	// Evmos keepers
	Erc20Keeper         erc20keeper.Keeper
	EpochsKeeper        epochskeeper.Keeper
	VestingKeeper       vestingkeeper.Keeper
	LiquidVestingKeeper liquidvestingkeeper.Keeper

	// Haqq keepers
	CoinomicsKeeper coinomicskeeper.Keeper
	DaoKeeper       ucdaokeeper.Keeper

	// the module manager
	mm                 *module.Manager
	BasicModuleManager module.BasicManager

	// the configurator
	configurator module.Configurator

	// simulation manager
	sm *module.SimulationManager

	// queryMultistore used on versionDB build
	qms storetypes.MultiStore

	tpsCounter *tpsCounter
}

// SimulationManager implements the SimulationApp interface
func (app *Haqq) SimulationManager() *module.SimulationManager {
	return app.sm
}

// NewHaqq returns a reference to a new initialized Ethermint application.
func NewHaqq(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *Haqq {
	encodingConfig := encoding.MakeConfig()
	appCodec := encodingConfig.Codec
	cdc := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	// setup memiavl if it's enabled in config (disabled for SDK v0.53.4 compatibility)
	// baseAppOptions = memiavlstore.SetupMemIAVL(logger, homePath, appOpts, false, false, baseAppOptions)

	// Setup Mempool and Proposal Handlers
	baseAppOptions = append(baseAppOptions, func(app *baseapp.BaseApp) {
		mp := mempool.NoOpMempool{}
		app.SetMempool(mp)
		handler := baseapp.NewDefaultProposalHandler(mp, app)
		app.SetPrepareProposal(handler.PrepareProposalHandler())
		app.SetProcessProposal(handler.ProcessProposalHandler())
	})

	// NOTE we use custom transaction decoder that supports the sdk.Tx interface instead of sdk.StdTx
	bApp := baseapp.NewBaseApp(
		Name,
		logger,
		db,
		encodingConfig.TxConfig.TxDecoder(),
		baseAppOptions...,
	)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	keys, memKeys, tkeys := StoreKeys()

	app := &Haqq{
		BaseApp:           bApp,
		cdc:               cdc,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// init params keeper and subspaces
	app.ParamsKeeper = initParamsKeeper(appCodec, cdc, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

	// get authority address
	authAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
		authAddr,
		runtime.EventService{},
	)
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])

	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	// scopedICAHostKeeper no longer required with ibc-go v10 ICA host keeper constructor

	// Applications that wish to enforce statically created ScopedKeepers should call `Seal` after creating
	// their scoped modules in `NewApp` with `ScopeToModule`
	app.CapabilityKeeper.Seal()

	// use custom Ethermint account for contracts
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		haqqtypes.ProtoAccount, maccPerms,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authAddr,
		authkeeper.WithUnorderedTransactions(true), // enable unordered transactions
	)
	haqqBankKeeper := haqqbankkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		&app.DistrKeeper,
		app.BlockedAddrs(),
		authAddr,
		logger,
	)
	app.BankKeeper = haqqBankKeeper.BaseKeeper

	// optional: enable sign mode textual by overwriting the default tx config (after setting the bank keeper)
	enabledSignModes := append(authtx.DefaultSignModes, sigtypes.SignMode_SIGN_MODE_TEXTUAL) //nolint:gocritic
	txConfigOpts := authtx.ConfigOptions{
		EnabledSignModes:           enabledSignModes,
		TextualCoinMetadataQueryFn: txmodule.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
	}
	txConfig, err := authtx.NewTxConfigWithOptions(
		appCodec,
		txConfigOpts,
	)
	if err != nil {
		panic(err)
	}
	app.txConfig = txConfig

	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		&haqqBankKeeper,
		authAddr,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		stakingKeeper,
		authtypes.FeeCollectorName,
		authAddr,
	)
	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		app.LegacyAmino(),
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		stakingKeeper,
		authAddr,
	)
	app.CrisisKeeper = *crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[crisistypes.StoreKey]),
		invCheckPeriod,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authAddr,
		app.AccountKeeper.AddressCodec(),
	)
	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[feegrant.StoreKey]), app.AccountKeeper)
	app.UpgradeKeeper = *upgradekeeper.NewKeeper(skipUpgradeHeights, runtime.NewKVStoreService(keys[upgradetypes.StoreKey]), appCodec, homePath, app.BaseApp, authAddr)

	app.AuthzKeeper = authzkeeper.NewKeeper(runtime.NewKVStoreService(keys[authzkeeper.StoreKey]), appCodec, app.MsgServiceRouter(), app.AccountKeeper)

	tracer := cast.ToString(appOpts.Get(srvflags.EVMTracer))

	// Create Ethermint keepers
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		appCodec, authtypes.NewModuleAddress(govtypes.ModuleName),
		keys[feemarkettypes.StoreKey],
		tkeys[feemarkettypes.TransientKey],
		app.GetSubspace(feemarkettypes.ModuleName),
	)

	evmKeeper := evmkeeper.NewKeeper(
		appCodec, keys[evmtypes.StoreKey], tkeys[evmtypes.TransientKey], authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, stakingKeeper, app.FeeMarketKeeper,
		// FIX: Temporary solution to solve keeper interdependency while new precompile module
		// is being developed.
		&app.Erc20Keeper,
		tracer, app.GetSubspace(evmtypes.ModuleName),
	)
	app.EvmKeeper = evmKeeper

	// Create IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[ibcexported.StoreKey]), app.GetSubspace(ibcexported.ModuleName), app.UpgradeKeeper, authAddr,
	)

	govConfig := govtypes.Config{
		MaxMetadataLen: 10000,
	}
	govKeeper := govkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[govtypes.StoreKey]), app.AccountKeeper, &haqqBankKeeper,
		stakingKeeper, app.DistrKeeper, app.MsgServiceRouter(), govConfig, authAddr,
	)

	// Haqq Keeper
	app.CoinomicsKeeper = coinomicskeeper.NewKeeper(
		keys[coinomicstypes.StoreKey], appCodec, app.GetSubspace(coinomicstypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, stakingKeeper,
		authtypes.FeeCollectorName,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	// NOTE: Distr, Slashing and Claim must be created before calling the Hooks method to avoid returning a Keeper without its table generated
	stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			app.DistrKeeper.Hooks(),
			app.SlashingKeeper.Hooks(),
		),
	)

	app.StakingKeeper = *stakingKeeper

	app.VestingKeeper = vestingkeeper.NewKeeper(
		keys[vestingtypes.StoreKey], appCodec,
		app.AccountKeeper, app.BankKeeper, app.DistrKeeper, app.StakingKeeper,
	)

	app.Erc20Keeper = erc20keeper.NewKeeper(
		keys[erc20types.StoreKey], appCodec, authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.EvmKeeper, app.StakingKeeper,
		app.AuthzKeeper, &app.TransferKeeper,
	)

	app.LiquidVestingKeeper = liquidvestingkeeper.NewKeeper(
		keys[liquidvestingtypes.StoreKey], appCodec, app.GetSubspace(liquidvestingtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, &app.Erc20Keeper, app.VestingKeeper,
	)

	app.DaoKeeper = ucdaokeeper.NewBaseKeeper(
		appCodec, keys[ucdaotypes.StoreKey], app.AccountKeeper, app.BankKeeper, authAddr,
	)

	// Initialize the packet forward middleware Keeper
	// It's important to note that the PFM Keeper must be initialized before the Transfer Keeper
	app.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[packetforwardtypes.StoreKey]),
		nil, // will be zero-value here, reference is set later on with SetTransferKeeper.
		app.IBCKeeper.ChannelKeeper,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.TransferKeeper = transferkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[ibctransfertypes.StoreKey]), app.GetSubspace(ibctransfertypes.ModuleName),
		app.PacketForwardKeeper, // ICS4 Wrapper: packet forwarding IBC middleware
		app.IBCKeeper.ChannelKeeper, bApp.MsgServiceRouter(),
		app.AccountKeeper, app.BankKeeper,
		app.Erc20Keeper, // Add ERC20 Keeper for ERC20 transfers
		authAddr,
	)

	epochsKeeper := epochskeeper.NewKeeper(appCodec, keys[epochstypes.StoreKey])
	app.EpochsKeeper = *epochsKeeper.SetHooks(
		epochskeeper.NewMultiEpochHooks(), // < insert epoch hooks receivers here
	)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(), // < insert gov hooks receivers here
	)

	// We call this after setting the hooks to ensure that the hooks are set on the keeper
	evmKeeper.WithStaticPrecompiles(
		evmkeeper.NewAvailableStaticPrecompiles(
			*stakingKeeper,
			app.DistrKeeper,
			app.BankKeeper,
			app.Erc20Keeper,
			app.VestingKeeper,
			app.AuthzKeeper,
			app.TransferKeeper,
			*app.IBCKeeper.ChannelKeeper,
		),
	)

	app.PacketForwardKeeper.SetTransferKeeper(app.TransferKeeper)

	// Override the ICS20 app module
	transferModule := transfer.NewAppModule(app.TransferKeeper)

	// Create the app.ICAHostKeeper
	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(app.keys[icahosttypes.StoreKey]),
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.AccountKeeper,
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		authAddr,
	)

	// create host IBC module
	icaHostIBCModule := icahost.NewIBCModule(app.ICAHostKeeper)

	// transfer stack contains (from top to bottom):
	// - Transfer
	// - Packet Forward Middleware
	// - ERC20 Middleware

	// create IBC module from bottom to top of stack
	var transferStack porttypes.IBCModule

	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = packetforward.NewIBCMiddleware(
		transferStack,
		app.PacketForwardKeeper,
		0,
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp, // forward timeout
	)
	transferStack = erc20.NewIBCMiddleware(app.Erc20Keeper, transferStack)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := porttypes.NewRouter()
	ibcRouter.
		AddRoute(icahosttypes.SubModuleName, icaHostIBCModule).
		AddRoute(ibctransfertypes.ModuleName, transferStack)

	app.IBCKeeper.SetRouter(ibcRouter)

	// Register Tendermint light client route (ibc-go v10)
	clientKeeper := app.IBCKeeper.ClientKeeper
	storeProvider := clientKeeper.GetStoreProvider()
	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
		&app.StakingKeeper,
		app.SlashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		// SDK app modules
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper,
			app, app.txConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		haqqbank.NewAppModule(
			appCodec,
			&haqqBankKeeper,
			app.AccountKeeper,
			app.GetSubspace(banktypes.ModuleName),
		),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(&app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, &app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(&app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),

		// ibc modules
		ibc.NewAppModule(app.IBCKeeper),
		ica.NewAppModule(nil, &app.ICAHostKeeper),
		// IBC light clients
		ibctm.NewAppModule(tmLightClientModule),
		transferModule,
		packetforward.NewAppModule(app.PacketForwardKeeper, app.GetSubspace(packetforwardtypes.ModuleName)),
		// Ethermint app modules
		evm.NewAppModule(app.EvmKeeper, app.AccountKeeper, app.GetSubspace(evmtypes.ModuleName)),
		feemarket.NewAppModule(app.FeeMarketKeeper, app.GetSubspace(feemarkettypes.ModuleName)),

		// Evmos app modules
		erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper, app.GetSubspace(erc20types.ModuleName)),
		epochs.NewAppModule(appCodec, app.EpochsKeeper),

		// Haqq app modules
		coinomics.NewAppModule(app.CoinomicsKeeper, app.AccountKeeper, *app.StakingKeeper.Keeper),
		vesting.NewAppModule(app.VestingKeeper, app.AccountKeeper, app.BankKeeper, *app.StakingKeeper.Keeper),
		liquidvesting.NewAppModule(appCodec, app.LiquidVestingKeeper, app.AccountKeeper, app.BankKeeper, &app.Erc20Keeper),
		ucdao.NewAppModule(appCodec, app.DaoKeeper, app.GetSubspace(ucdaotypes.ModuleName)),
	)

	// BasicModuleManager defines the module BasicManager which is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default, it is composed of all the modules from the module manager.
	// Additionally, app module basics can be overwritten by passing them as an argument.
	app.BasicModuleManager = module.NewBasicManagerFromManager(
		app.mm,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
			stakingtypes.ModuleName: staking.AppModuleBasic{AppModuleBasic: &sdkstaking.AppModuleBasic{}},
			govtypes.ModuleName: gov.NewAppModuleBasic(
				[]govclient.ProposalHandler{
					paramsclient.ProposalHandler,
				},
			),
			ibctransfertypes.ModuleName: transfer.AppModuleBasic{AppModuleBasic: &ibctransfer.AppModuleBasic{}},
		},
	)
	app.BasicModuleManager.RegisterLegacyAminoCodec(cdc)
	app.BasicModuleManager.RegisterInterfaces(interfaceRegistry)

	// NOTE: upgrade module is required to be prioritized
	// In SDK v0.53.4, all registered modules must be listed in PreBlockers order,
	// even if they don't implement PreBlocker
	app.mm.SetOrderPreBlockers(
		upgradetypes.ModuleName,
		genutiltypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		capabilitytypes.ModuleName,
		crisistypes.ModuleName,
		govtypes.ModuleName,
		slashingtypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		evidencetypes.ModuleName,
		paramstypes.ModuleName,
		feegrant.ModuleName,
		authz.ModuleName,
		consensusparamtypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibctm.ModuleName,
		ibctransfertypes.ModuleName,
		packetforwardtypes.ModuleName,
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		erc20types.ModuleName,
		epochstypes.ModuleName,
		coinomicstypes.ModuleName,
		vestingtypes.ModuleName,
		liquidvestingtypes.ModuleName,
		ucdaotypes.ModuleName,
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0.
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	app.mm.SetOrderBeginBlockers(
		capabilitytypes.ModuleName,
		// Note: epochs' begin should be "real" start of epochs, we keep epochs beginblock at the beginning
		epochstypes.ModuleName,
		feemarkettypes.ModuleName,
		evmtypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		authz.ModuleName,
		// feegrant.ModuleName, // doesn't have begin blocker
	)

	// NOTE: fee market module must go last in order to retrieve the block gas used.
	app.mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		evmtypes.ModuleName,
		coinomicstypes.ModuleName,
		feemarkettypes.ModuleName,
		feegrant.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		// SDK modules
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		ibcexported.ModuleName,
		// Ethermint modules
		evmtypes.ModuleName,
		// NOTE: feemarket module needs to be initialized before genutil module:
		// gentx transactions use MinGasPriceDecorator.AnteHandle
		feemarkettypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		packetforwardtypes.ModuleName,
		icatypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		// Haqq modules
		vestingtypes.ModuleName,
		liquidvestingtypes.ModuleName,
		coinomicstypes.ModuleName,
		erc20types.ModuleName,
		epochstypes.ModuleName,
		ucdaotypes.ModuleName,
		// NOTE: crisis module must go at the end to check for invariants on each module
		crisistypes.ModuleName,
		consensusparamtypes.ModuleName,
	)

	app.mm.RegisterInvariants(&app.CrisisKeeper)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	err = app.mm.RegisterServices(app.configurator)
	if err != nil {
		panic(err)
	}

	// add test gRPC service for testing gRPC queries in isolation
	// testdata.RegisterTestServiceServer(app.GRPCQueryRouter(), testdata.TestServiceImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.mm.Modules, overrideModules)

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.mm.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)
	// add test gRPC service for testing gRPC queries in isolation
	testdata_pulsar.RegisterQueryServer(app.GRPCQueryRouter(), testdata_pulsar.QueryImpl{})

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// load state streaming if enabled
	if err := app.RegisterStreamingServices(appOpts, keys); err != nil {
		fmt.Printf("failed to load state streaming: %s", err)
		os.Exit(1)
	}

	// wire up the versiondb's `StreamingService` and `MultiStore`.
	if cast.ToBool(appOpts.Get("versiondb.enable")) {
		app.qms, err = app.setupVersionDB(homePath, keys, tkeys, memKeys)
		//nolint:nolintlint
		if err != nil {
			panic(errorsmod.Wrap(err, "error on versionDB setup"))
		}
	}

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)

	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))

	app.setAnteHandler(app.txConfig, maxGasWanted)
	app.setPostHandler()
	app.SetEndBlocker(app.EndBlocker)
	app.setupUpgradeHandlers()

	// At startup, after all modules have been registered, check that all prot
	// annotations are correct.
	protoFiles, err := proto.MergedRegistry()
	if err != nil {
		panic(err)
	}
	err = msgservice.ValidateProtoAnnotations(protoFiles)
	if err != nil {
		// Once we switch to using protoreflect-based antehandlers, we might
		// want to panic here instead of logging a warning.
		fmt.Fprintln(os.Stderr, err.Error())
	}

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			logger.Error("error on loading last version", "err", err)
			os.Exit(1)
		}

		// queryMultiStore will be only defined when using versionDB
		// when defined, we check if the iavl & versionDB versions match
		if app.qms != nil {
			v1 := app.qms.LatestVersion()
			v2 := app.LastBlockHeight()
			// Prevent creating gaps in versiondb
			// - if versiondb lag behind iavl, when commit new blocks, it creates gap in versiondb.
			// 	 This can happen because cms is committed before versiondb.
			// - if versiondb is beyond iavl, and when commit new blocks, versiondb will write some duplicated data.
			//	 This is actually not harmful, if the rewritten data is identical to the old ones.
			// 	 This can happen with memiavl async-commit.
			// The latter case is not harmful, so we can relax the checking to improve UX.
			if v1 > 0 && v1 < v2 {
				tmos.Exit(fmt.Sprintf("latest version of version-db (%d) lags behind iavl's latest version (%d)", v1, v2))
			}
		}
	}

	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper

	// Finally start the tpsCounter.
	app.tpsCounter = newTPSCounter(logger)
	go func() {
		// Unfortunately golangci-lint is so pedantic
		// so we have to ignore this error explicitly.
		_ = app.tpsCounter.start(context.Background())
	}()

	return app
}

// Name returns the name of the App
func (app *Haqq) Name() string { return app.BaseApp.Name() }

func (app *Haqq) setAnteHandler(txConfig client.TxConfig, maxGasWanted uint64) {
	options := ante.HandlerOptions{
		Cdc:                    app.appCodec,
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		ExtensionOptionChecker: haqqtypes.HasDynamicFeeExtensionOption,
		EvmKeeper:              app.EvmKeeper,
		StakingKeeper:          app.StakingKeeper,
		FeegrantKeeper:         app.FeeGrantKeeper,
		DistributionKeeper:     app.DistrKeeper,
		IBCKeeper:              app.IBCKeeper,
		FeeMarketKeeper:        app.FeeMarketKeeper,
		SignModeHandler:        txConfig.SignModeHandler(),
		SigGasConsumer:         ante.SigVerificationGasConsumer,
		MaxTxGasWanted:         maxGasWanted,
		TxFeeChecker:           ethante.NewDynamicFeeChecker(app.EvmKeeper),
		SigVerifyOptions: []authante.SigVerificationDecoratorOption{
			// Configure unordered transaction gas cost (default: 2240)
			authante.WithUnorderedTxGasCost(authante.DefaultUnorderedTxGasCost),
			// Configure maximum unordered transaction timeout duration (default: 10 minutes)
			authante.WithMaxUnorderedTxTimeoutDuration(authante.DefaultMaxTimeoutDuration),
		},
	}

	if err := options.Validate(); err != nil {
		panic(err)
	}

	app.SetAnteHandler(ante.NewCommunityPoolSpendAnteHandler(ante.NewAnteHandler(options)))
}

func (app *Haqq) setPostHandler() {
	postHandler, err := posthandler.NewPostHandler(
		posthandler.HandlerOptions{},
	)
	if err != nil {
		panic(err)
	}
	app.SetPostHandler(postHandler)
}

// BeginBlocker runs the CometBFT ABCI BeginBlock logic. It executes state changes at the beginning
// of the new block for every registered module. If there is a registered fork at the current height,
// BeginBlocker will schedule the upgrade plan and perform the state migration (if any).
func (app *Haqq) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.mm.BeginBlock(ctx)
}

// EndBlocker updates every end block
func (app *Haqq) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.mm.EndBlock(ctx)
}

// The DeliverTx method is intentionally decomposed to calculate the transactions per second.
func (app *Haqq) FinalizeBlock(req *abci.RequestFinalizeBlock) (res *abci.ResponseFinalizeBlock, err error) {
	defer func() {
		// TODO: Record the count along with the code and or reason so as to display
		// in the transactions per second live dashboards.
		if res == nil || err != nil {
			return
		}
		for _, txRes := range res.TxResults {
			if txRes.IsErr() {
				app.tpsCounter.incrementFailure()
			} else {
				app.tpsCounter.incrementSuccess()
			}
		}
	}()
	res, err = app.BaseApp.FinalizeBlock(req)
	return
}

// InitChainer updates at chain initialization
func (app *Haqq) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState haqqtypes.GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap()); err != nil {
		panic(err)
	}

	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

func (app *Haqq) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.mm.PreBlock(ctx)
}

// LoadHeight loads state at a particular height
func (app *Haqq) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *Haqq) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)

	accs := make([]string, 0, len(maccPerms))
	for k := range maccPerms {
		accs = append(accs, k)
	}
	sort.Strings(accs)

	for _, acc := range accs {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedAddrs returns all the addresses that are not
// allowed to receive external tokens.
//
// These include:
//   - module accounts
//   - Ethereum's native precompiles
//   - the static precompiled contracts available through evmOS
func (app *Haqq) BlockedAddrs() map[string]bool {
	blockedAddrs := make(map[string]bool)

	accs := make([]string, 0, len(maccPerms))
	for k := range maccPerms {
		accs = append(accs, k)
	}
	sort.Strings(accs)

	for _, acc := range accs {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	blockedPrecompilesHex := evmtypes.AvailableStaticPrecompiles
	for _, addr := range vm.PrecompiledAddressesBerlin {
		blockedPrecompilesHex = append(blockedPrecompilesHex, addr.Hex())
	}

	for _, precompile := range blockedPrecompilesHex {
		blockedAddrs[utils.GetAccAddressFromEthAddress(precompile).String()] = true
	}

	return blockedAddrs
}

// LegacyAmino returns Haqq's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *Haqq) LegacyAmino() *codec.LegacyAmino {
	return app.cdc
}

// AppCodec returns Haqq's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *Haqq) AppCodec() codec.Codec {
	return app.appCodec
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *Haqq) DefaultGenesis() haqqtypes.GenesisState {
	return app.BasicModuleManager.DefaultGenesis(app.appCodec)
}

// InterfaceRegistry returns Evmos's InterfaceRegistry
func (app *Haqq) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *Haqq) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *Haqq) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *Haqq) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *Haqq) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *Haqq) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new CometBFT queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register node gRPC service for grpc-gateway.
	node.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	app.BasicModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
}

func (app *Haqq) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *Haqq) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// RegisterNodeService registers the node gRPC service on the provided
// application gRPC query router.
func (app *Haqq) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// IBC Go TestingApp functions

// GetBaseApp implements the TestingApp interface.
func (app *Haqq) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetStakingKeeperSDK implements the TestingApp interface.
func (app *Haqq) GetStakingKeeperSDK() stakingkeeper.Keeper {
	return app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *Haqq) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *Haqq) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// GetTxConfig implements the TestingApp interface.
func (app *Haqq) GetTxConfig() client.TxConfig {
	return app.txConfig
}

// AutoCliOpts returns the autocli options for the app.
func (app *Haqq) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule)
	for _, m := range app.mm.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(app.mm.Modules),
		AddressCodec:          authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// SDK subspaces
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	// IBC subspaces
	keyTable := ibcclienttypes.ParamKeyTable()
	keyTable.RegisterParamSet(&ibcconnectiontypes.Params{})
	paramsKeeper.Subspace(ibcexported.ModuleName).WithKeyTable(keyTable)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName).WithKeyTable(ibctransfertypes.ParamKeyTable())
	paramsKeeper.Subspace(icahosttypes.SubModuleName).WithKeyTable(icahosttypes.ParamKeyTable())
	paramsKeeper.Subspace(packetforwardtypes.ModuleName)
	// ethermint subspaces
	paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable())
	paramsKeeper.Subspace(feemarkettypes.ModuleName).WithKeyTable(feemarkettypes.ParamKeyTable())
	// evmos subspaces
	paramsKeeper.Subspace(erc20types.ModuleName)
	// haqq subspaces
	paramsKeeper.Subspace(coinomicstypes.ModuleName)
	paramsKeeper.Subspace(liquidvestingtypes.ModuleName)
	paramsKeeper.Subspace(ucdaotypes.ModuleName)

	return paramsKeeper
}

func (app *Haqq) setupUpgradeHandlers() {
	// v1.10.0 Upgrade Cosmos SDK to v0.53.4 and IBC to v10
	// Note: PreBlockers order already includes authtypes.ModuleName as required by v0.53.x
	app.UpgradeKeeper.SetUpgradeHandler(
		v1100.UpgradeName,
		v1100.CreateUpgradeHandler(app.mm, app.configurator),
	)

	// v1.9.1 Upgrade Cosmos SDK to v0.50.9 and IBC to v8.6.1 with minor fixes for ERC20
	app.UpgradeKeeper.SetUpgradeHandler(
		v191.UpgradeName,
		v191.CreateUpgradeHandler(app.mm, app.configurator, app.GovKeeper, app.Erc20Keeper),
	)

	// When a planned update height is reached, the old binary will panic
	// writing on disk the height and name of the update that triggered it
	// This will read that value, and execute the preparations for the upgrade.
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Errorf("failed to read upgrade info from disk: %w", err))
	}

	if upgradeInfo.Name == v1100.UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Deleted: []string{"capability", "feeibc"},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
