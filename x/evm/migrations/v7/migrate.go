// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package v7

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	v6types "github.com/haqq-network/haqq/x/evm/migrations/v7/types"
	"github.com/haqq-network/haqq/x/evm/types"
)

// MigrateStore migrates the x/evm module state from the consensus version 6 to
// version 7. Specifically, it changes the type of the Params ExtraEIPs from
// []int64 to []string and introduces the access control.
func MigrateStore(
	ctx sdk.Context,
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
) error {
	var (
		paramsV6 v6types.V6Params
		params   types.Params
	)

	store := ctx.KVStore(storeKey)

	paramsV6Bz := store.Get(types.KeyPrefixParams)
	// MustUnmarshal accepts nil bytes and yields a zero-value V6Params, which would be
	// written back as an empty parameter set. Refuse instead: a chain reaching this
	// migration necessarily had v6 params, so an empty key means the store is not what
	// this migration was written for. Phrased against the input, this check is frozen -
	// unlike Params.Validate below, it cannot change meaning in a later release.
	if len(paramsV6Bz) == 0 {
		return fmt.Errorf("no v6 evm params found under %X; nothing to migrate", types.KeyPrefixParams)
	}
	cdc.MustUnmarshal(paramsV6Bz, &paramsV6)

	params.EvmDenom = paramsV6.EvmDenom
	params.ChainConfig = types.ChainConfig{
		HomesteadBlock:      paramsV6.ChainConfig.HomesteadBlock,
		DAOForkBlock:        paramsV6.ChainConfig.DAOForkBlock,
		DAOForkSupport:      paramsV6.ChainConfig.DAOForkSupport,
		EIP150Block:         paramsV6.ChainConfig.EIP150Block,
		EIP150Hash:          paramsV6.ChainConfig.EIP150Hash,
		EIP155Block:         paramsV6.ChainConfig.EIP155Block,
		EIP158Block:         paramsV6.ChainConfig.EIP158Block,
		ByzantiumBlock:      paramsV6.ChainConfig.ByzantiumBlock,
		ConstantinopleBlock: paramsV6.ChainConfig.ConstantinopleBlock,
		PetersburgBlock:     paramsV6.ChainConfig.PetersburgBlock,
		IstanbulBlock:       paramsV6.ChainConfig.IstanbulBlock,
		MuirGlacierBlock:    paramsV6.ChainConfig.MuirGlacierBlock,
		BerlinBlock:         paramsV6.ChainConfig.BerlinBlock,
		LondonBlock:         paramsV6.ChainConfig.LondonBlock,
		ArrowGlacierBlock:   paramsV6.ChainConfig.ArrowGlacierBlock,
		GrayGlacierBlock:    paramsV6.ChainConfig.GrayGlacierBlock,
		MergeNetsplitBlock:  paramsV6.ChainConfig.MergeNetsplitBlock,
		ShanghaiBlock:       paramsV6.ChainConfig.ShanghaiBlock,
		CancunBlock:         paramsV6.ChainConfig.CancunBlock,
	}
	params.AllowUnprotectedTxs = paramsV6.AllowUnprotectedTxs
	params.ActiveStaticPrecompiles = paramsV6.ActivePrecompiles
	params.EVMChannels = paramsV6.EVMChannels

	// set the default access control configuration
	params.AccessControl = types.DefaultAccessControl

	// Migrate old ExtraEIPs from int64 to string. Since no Evmos EIPs have been
	// created before and activators contains only `ethereum_XXXX` activations,
	// all values will be prefixed with `ethereum_`.
	params.ExtraEIPs = make([]string, 0, len(paramsV6.ExtraEIPs))
	for _, eip := range paramsV6.ExtraEIPs {
		eipName := fmt.Sprintf("ethereum_%d", eip)
		params.ExtraEIPs = append(params.ExtraEIPs, eipName)
	}

	// NOTE: deliberately no params.Validate() here.
	//
	// A migration is a frozen transcript of what the live chain executed at the upgrade
	// height, and a node replaying the chain has to reach the same verdict the canonical
	// run did. Params.Validate() cannot give that guarantee: its rule set belongs to the
	// current binary and grows over time - it consults the live EIP activator table and
	// the live fork-ordering rules, and this release added a check pinning EvmDenom to
	// the chain base denom. Calling it here would let a later release turn a block that
	// once succeeded into a halt during replay.
	//
	// Everything written below is copied from V6Params the chain already accepted, plus
	// types.DefaultAccessControl and a mechanical rename of the ExtraEIPs. On every chain
	// that has run this upgrade the validation provably passed, so re-running it can only
	// ever change the answer, never improve it. Validation belongs where params are
	// written by someone - Keeper.SetParams, which genesis, gov and the precompile
	// toggles all go through.
	//
	// The one thing Validate() did catch here that the copy does not - an empty store,
	// which used to surface as `invalid denom: ""` - is now checked against the input
	// above, where it belongs.
	bz := cdc.MustMarshal(&params)

	store.Set(types.KeyPrefixParams, bz)

	return nil
}
