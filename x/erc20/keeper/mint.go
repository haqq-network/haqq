package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	erc20types "github.com/evmos/evmos/v14/x/erc20/types"
)

// MintingEnabled checks that:
//   - the global parameter for erc20 conversion is enabled
//   - minting is enabled for the given (erc20,coin) token pair
//   - recipient address is not on the blocked list
//   - bank module transfers are enabled for the Cosmos coin
func (k Keeper) MintingEnabled(
	ctx sdk.Context,
	sender, receiver sdk.AccAddress,
	token string,
) (erc20types.TokenPair, error) {
	if !k.IsERC20Enabled(ctx) {
		return erc20types.TokenPair{}, errorsmod.Wrap(
			erc20types.ErrERC20Disabled, "module is currently disabled by governance",
		)
	}

	id := k.GetTokenPairID(ctx, token)
	if len(id) == 0 {
		return erc20types.TokenPair{}, errorsmod.Wrapf(
			erc20types.ErrTokenPairNotFound, "token '%s' not registered by id", token,
		)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return erc20types.TokenPair{}, errorsmod.Wrapf(
			erc20types.ErrTokenPairNotFound, "token '%s' not registered", token,
		)
	}

	if !pair.Enabled {
		return erc20types.TokenPair{}, errorsmod.Wrapf(
			erc20types.ErrERC20TokenPairDisabled, "minting token '%s' is not enabled by governance", token,
		)
	}

	if k.bankKeeper.BlockedAddr(receiver.Bytes()) {
		return erc20types.TokenPair{}, errorsmod.Wrapf(
			errortypes.ErrUnauthorized, "%s is not allowed to receive transactions", receiver,
		)
	}

	// NOTE: ignore amount as only denom is checked on IsSendEnabledCoin
	coin := sdk.Coin{Denom: pair.Denom}

	// check if minting to a recipient address other than the sender is enabled
	// for for the given coin denom
	if !sender.Equals(receiver) && !k.bankKeeper.IsSendEnabledCoin(ctx, coin) {
		return erc20types.TokenPair{}, errorsmod.Wrapf(
			banktypes.ErrSendDisabled, "minting '%s' coins to an external address is currently disabled", token,
		)
	}

	return pair, nil
}
