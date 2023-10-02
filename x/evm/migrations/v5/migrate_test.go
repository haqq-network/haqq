package v5_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v14/encoding"

	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	"github.com/haqq-network/haqq/app"
	v5 "github.com/haqq-network/haqq/x/evm/migrations/v5"
	v5types "github.com/haqq-network/haqq/x/evm/migrations/v5/types"
)

func TestMigrate(t *testing.T) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)
	cdc := encCfg.Codec

	storeKey := sdk.NewKVStoreKey(evmtypes.ModuleName)
	tKey := sdk.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	kvStore := ctx.KVStore(storeKey)

	extraEIPs := v5types.V5ExtraEIPs{EIPs: evmtypes.AvailableExtraEIPs}
	extraEIPsBz := cdc.MustMarshal(&extraEIPs)
	chainConfig := evmtypes.DefaultChainConfig()
	chainConfigBz := cdc.MustMarshal(&chainConfig)

	// Set the params in the store
	kvStore.Set(evmtypes.ParamStoreKeyEVMDenom, []byte(evmtypes.DefaultEVMDenom))
	kvStore.Set(evmtypes.ParamStoreKeyEnableCreate, []byte{0x01})
	kvStore.Set(evmtypes.ParamStoreKeyEnableCall, []byte{0x01})
	kvStore.Set(evmtypes.ParamStoreKeyAllowUnprotectedTxs, []byte{0x01})
	kvStore.Set(evmtypes.ParamStoreKeyExtraEIPs, extraEIPsBz)
	kvStore.Set(evmtypes.ParamStoreKeyChainConfig, chainConfigBz)

	err := v5.MigrateStore(ctx, storeKey, cdc)
	require.NoError(t, err)

	paramsBz := kvStore.Get(evmtypes.KeyPrefixParams)
	var params evmtypes.Params
	cdc.MustUnmarshal(paramsBz, &params)

	// test that the params have been migrated correctly
	require.Equal(t, evmtypes.DefaultEVMDenom, params.EvmDenom)
	require.True(t, params.EnableCreate)
	require.True(t, params.EnableCall)
	require.True(t, params.AllowUnprotectedTxs)
	require.Equal(t, chainConfig, params.ChainConfig)
	require.Equal(t, extraEIPs.EIPs, params.ExtraEIPs)

	// check that the keys are deleted
	require.False(t, kvStore.Has(evmtypes.ParamStoreKeyEVMDenom))
	require.False(t, kvStore.Has(evmtypes.ParamStoreKeyEnableCreate))
	require.False(t, kvStore.Has(evmtypes.ParamStoreKeyEnableCall))
	require.False(t, kvStore.Has(evmtypes.ParamStoreKeyAllowUnprotectedTxs))
	require.False(t, kvStore.Has(evmtypes.ParamStoreKeyExtraEIPs))
	require.False(t, kvStore.Has(evmtypes.ParamStoreKeyChainConfig))
}
