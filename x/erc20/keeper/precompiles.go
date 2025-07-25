// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/erc20"
	"github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

// GetERC20PrecompileInstance returns the precompile instance for the given address.
func (k Keeper) GetERC20PrecompileInstance(
	ctx sdk.Context,
	address common.Address,
) (contract vm.PrecompiledContract, found bool, err error) {
	params := k.GetParams(ctx)

	if k.IsAvailableERC20Precompile(&params, address) {
		precompile, err := k.InstantiateERC20Precompile(ctx, address)
		if err != nil {
			return nil, false, errorsmod.Wrapf(err, "precompiled contract not initialized: %s", address.String())
		}
		return precompile, true, nil
	}
	return nil, false, nil
}

// InstantiateERC20Precompile returns an ERC20 precompile instance for the given contract address
func (k Keeper) InstantiateERC20Precompile(ctx sdk.Context, contractAddr common.Address) (vm.PrecompiledContract, error) {
	address := contractAddr.String()
	// check if the precompile is an ERC20 contract
	id := k.GetTokenPairID(ctx, address)
	if len(id) == 0 {
		return nil, fmt.Errorf("precompile id not found: %s", address)
	}
	pair, ok := k.GetTokenPair(ctx, id)
	if !ok {
		return nil, fmt.Errorf("token pair not found: %s", address)
	}
	return erc20.NewPrecompile(pair, k.bankKeeper, k.authzKeeper, *k.transferKeeper)
}

// IsAvailableERC20Precompile returns true if the given precompile address is contained in the
// EVM keeper's available dynamic precompiles params.
func (k Keeper) IsAvailableERC20Precompile(params *types.Params, address common.Address) bool {
	return params.IsNativePrecompile(address) ||
		params.IsDynamicPrecompile(address)
}
