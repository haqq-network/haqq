package v4_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v14/encoding"

	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	"github.com/haqq-network/haqq/app"
	v4 "github.com/haqq-network/haqq/x/evm/migrations/v4"
	v4types "github.com/haqq-network/haqq/x/evm/migrations/v4/types"
)

type mockSubspace struct {
	ps evmtypes.Params
}

func newMockSubspace(ps evmtypes.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSetIfExists(_ sdk.Context, ps evmtypes.LegacyParams) {
	*ps.(*evmtypes.Params) = ms.ps
}

func TestMigrate(t *testing.T) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(evmtypes.ModuleName)
	tKey := sdk.NewTransientStoreKey(evmtypes.TransientKey)
	ctx := testutil.DefaultContext(storeKey, tKey)
	kvStore := ctx.KVStore(storeKey)

	legacySubspace := newMockSubspace(evmtypes.DefaultParams())
	require.NoError(t, v4.MigrateStore(ctx, storeKey, legacySubspace, cdc))

	// Get all the new parameters from the kvStore
	var evmDenom string
	bz := kvStore.Get(evmtypes.ParamStoreKeyEVMDenom)
	evmDenom = string(bz)

	allowUnprotectedTx := kvStore.Has(evmtypes.ParamStoreKeyAllowUnprotectedTxs)
	enableCreate := kvStore.Has(evmtypes.ParamStoreKeyEnableCreate)
	enableCall := kvStore.Has(evmtypes.ParamStoreKeyEnableCall)

	var chainCfg v4types.V4ChainConfig
	bz = kvStore.Get(evmtypes.ParamStoreKeyChainConfig)
	cdc.MustUnmarshal(bz, &chainCfg)

	var extraEIPs v4types.ExtraEIPs
	bz = kvStore.Get(evmtypes.ParamStoreKeyExtraEIPs)
	cdc.MustUnmarshal(bz, &extraEIPs)
	require.Equal(t, []int64(nil), extraEIPs.EIPs)

	params := v4types.V4Params{
		EvmDenom:            evmDenom,
		AllowUnprotectedTxs: allowUnprotectedTx,
		EnableCreate:        enableCreate,
		EnableCall:          enableCall,
		V4ChainConfig:       chainCfg,
		ExtraEIPs:           extraEIPs,
	}

	require.Equal(t, legacySubspace.ps.EnableCall, params.EnableCall)
	require.Equal(t, legacySubspace.ps.EnableCreate, params.EnableCreate)
	require.Equal(t, legacySubspace.ps.AllowUnprotectedTxs, params.AllowUnprotectedTxs)
	require.Equal(t, legacySubspace.ps.ExtraEIPs, params.ExtraEIPs.EIPs)
	require.EqualValues(t, legacySubspace.ps.ChainConfig, params.V4ChainConfig)
}
