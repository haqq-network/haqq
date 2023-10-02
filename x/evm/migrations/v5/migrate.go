package v5

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	v5types "github.com/haqq-network/haqq/x/evm/migrations/v5/types"
)

// MigrateStore migrates the x/evm module state from the consensus version 4 to
// version 5. Specifically, it takes the parameters that are currently stored
// in separate keys and stores them directly into the x/evm module state using
// a single params key.
func MigrateStore(
	ctx sdk.Context,
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
) error {
	var (
		extraEIPs   v5types.V5ExtraEIPs
		chainConfig evmtypes.ChainConfig
		params      evmtypes.Params
	)

	store := ctx.KVStore(storeKey)

	denom := string(store.Get(evmtypes.ParamStoreKeyEVMDenom))

	extraEIPsBz := store.Get(evmtypes.ParamStoreKeyExtraEIPs)
	cdc.MustUnmarshal(extraEIPsBz, &extraEIPs)

	chainCfgBz := store.Get(evmtypes.ParamStoreKeyChainConfig)
	cdc.MustUnmarshal(chainCfgBz, &chainConfig)

	params.EvmDenom = denom
	params.ExtraEIPs = extraEIPs.EIPs
	params.ChainConfig = chainConfig
	params.EnableCreate = store.Has(evmtypes.ParamStoreKeyEnableCreate)
	params.EnableCall = store.Has(evmtypes.ParamStoreKeyEnableCall)
	params.AllowUnprotectedTxs = store.Has(evmtypes.ParamStoreKeyAllowUnprotectedTxs)

	store.Delete(evmtypes.ParamStoreKeyChainConfig)
	store.Delete(evmtypes.ParamStoreKeyExtraEIPs)
	store.Delete(evmtypes.ParamStoreKeyEVMDenom)
	store.Delete(evmtypes.ParamStoreKeyEnableCreate)
	store.Delete(evmtypes.ParamStoreKeyEnableCall)
	store.Delete(evmtypes.ParamStoreKeyAllowUnprotectedTxs)

	if err := params.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&params)

	store.Set(evmtypes.KeyPrefixParams, bz)
	return nil
}
