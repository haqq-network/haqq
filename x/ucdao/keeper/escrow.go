package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k BaseKeeper) escrowToken(ctx sdk.Context, sender, escrowAddress sdk.AccAddress, coin sdk.Coin) error {
	if err := k.bk.SendCoins(ctx, sender, escrowAddress, sdk.NewCoins(coin)); err != nil {
		// failure is expected for insufficient balances
		return err
	}

	// track the total amount in escrow keyed by denomination to allow for efficient iteration
	currentTotalEscrow := k.GetTotalBalanceOf(ctx, coin.Denom)
	newTotalEscrow := currentTotalEscrow.Add(coin)
	k.setTotalBalanceOfCoin(ctx, newTotalEscrow)

	return nil
}

func (k BaseKeeper) unescrowToken(ctx sdk.Context, escrowAddress, receiver sdk.AccAddress, coin sdk.Coin) error { //nolint: all
	if err := k.bk.SendCoins(ctx, escrowAddress, receiver, sdk.NewCoins(coin)); err != nil {
		// NOTE: this error is only expected to occur given an unexpected bug or a malicious
		// counterparty module. The bug may occur in bank or any part of the code that allows
		// the escrow address to be drained. A malicious counterparty module could drain the
		// escrow address by allowing more tokens to be sent back then were escrowed.
		return sdkerrors.Wrap(err, "unable to unescrow tokens, this may be caused by a malicious counterparty module or a bug: please open an issue on counterparty module")
	}

	// track the total amount in escrow keyed by denomination to allow for efficient iteration
	currentTotalEscrow := k.GetTotalBalanceOf(ctx, coin.Denom)
	newTotalEscrow := currentTotalEscrow.Sub(coin)
	k.setTotalBalanceOfCoin(ctx, newTotalEscrow)

	return nil
}

func (k BaseKeeper) transferEscrowToken(ctx sdk.Context, escrowAddress, newEscrowAddress sdk.AccAddress, token sdk.Coin) error {
	if err := k.bk.SendCoins(ctx, escrowAddress, newEscrowAddress, sdk.NewCoins(token)); err != nil {
		// NOTE: this error is only expected to occur given an unexpected bug or a malicious
		// counterparty module. The bug may occur in bank or any part of the code that allows
		// the escrow address to be drained. A malicious counterparty module could drain the
		// escrow address by allowing more tokens to be sent back then were escrowed.
		return sdkerrors.Wrap(err, "unable to transfer escrow tokens, this may be caused by a malicious counterparty module or a bug: please open an issue on counterparty module")
	}

	return nil
}
