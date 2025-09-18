package app

import (
	storetypes "cosmossdk.io/store/types"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	epochstypes "github.com/haqq-network/haqq/x/epochs/types"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// StoreKeys returns the application store keys,
// the EVM transient store keys and the memory store keys
func StoreKeys() (
	map[string]*storetypes.KVStoreKey,
	map[string]*storetypes.MemoryStoreKey,
	map[string]*storetypes.TransientStoreKey,
) {
	keys := storetypes.NewKVStoreKeys(
		// SDK keys
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, upgradetypes.StoreKey,
		evidencetypes.StoreKey, capabilitytypes.StoreKey, consensusparamtypes.StoreKey,
		feegrant.StoreKey, authzkeeper.StoreKey, crisistypes.StoreKey,
		// ibc keys
		ibcexported.StoreKey, ibctransfertypes.StoreKey,
		packetforwardtypes.StoreKey,
		// ica keys
		icahosttypes.StoreKey,
		// ethermint keys
		evmtypes.StoreKey, feemarkettypes.StoreKey,
		// evmos keys
		erc20types.StoreKey,
		epochstypes.StoreKey, vestingtypes.StoreKey,
		// haqq keys
		coinomicstypes.StoreKey,
		liquidvestingtypes.StoreKey,
		ucdaotypes.StoreKey,
	)

	// Add the EVM transient store key
	memKeys := storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey, evmtypes.TransientKey, feemarkettypes.TransientKey)

	return keys, memKeys, tkeys
}
