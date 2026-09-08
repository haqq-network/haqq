package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TrackAddBalance update internal total balance value.
// NOTE: This method intentionally has been exported as such feature required by ethiq module.
func (k BaseKeeper) TrackAddBalance(ctx sdk.Context, coin sdk.Coin) {
	currentTotalEscrow := k.GetTotalBalanceOf(ctx, coin.Denom)
	newTotalEscrow := currentTotalEscrow.Add(coin)
	k.setTotalBalanceOfCoin(ctx, newTotalEscrow)
}

// TrackSubBalance update internal total balance value.
// NOTE: This method intentionally has been exported as such feature required by ethiq module.
//
// The subtraction saturates at zero. The total balance counts only what entered through Fund,
// TransferOwnership or genesis, while the escrow balances it is supposed to mirror live in bank
// and accept plain transfers from anyone (see GetHolders). Burning such an escrow - through
// ConvertToHaqq or an ethiq application - then asks to remove more than was ever counted, and
// sdk.Coin.Sub panics on a negative result. A panic inside a consensus path is worse than an
// understated counter, so clamp and record the discrepancy instead.
//
// This bounds the damage, it does not fix it: the counter is a second source of truth that
// cannot be kept honest while escrows accept untracked transfers.
func (k BaseKeeper) TrackSubBalance(ctx sdk.Context, coin sdk.Coin) {
	currentTotalEscrow := k.GetTotalBalanceOf(ctx, coin.Denom)

	if currentTotalEscrow.IsLT(coin) {
		k.Logger(ctx).Error(
			"ucdao total balance is lower than the amount being removed; clamping to zero",
			"denom", coin.Denom,
			"requested", coin.Amount.String(),
			"tracked", currentTotalEscrow.Amount.String(),
		)
		k.setTotalBalanceOfCoin(ctx, sdk.NewCoin(coin.Denom, math.ZeroInt()))
		return
	}

	newTotalEscrow := currentTotalEscrow.Sub(coin)
	k.setTotalBalanceOfCoin(ctx, newTotalEscrow)
}

func (k BaseKeeper) escrowToken(ctx sdk.Context, sender, escrowAddress sdk.AccAddress, coin sdk.Coin) error {
	if err := k.bk.SendCoins(ctx, sender, escrowAddress, sdk.NewCoins(coin)); err != nil {
		// failure is expected for insufficient balances
		return err
	}

	// track the total amount in escrow keyed by denomination to allow for efficient iteration
	k.TrackAddBalance(ctx, coin)

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
	k.TrackSubBalance(ctx, coin)

	return nil
}

func (k BaseKeeper) transferEscrowToken(ctx sdk.Context, escrowAddress, newEscrowAddress sdk.AccAddress, amt sdk.Coins) error {
	if err := k.bk.SendCoins(ctx, escrowAddress, newEscrowAddress, amt); err != nil {
		// NOTE: this error is only expected to occur given an unexpected bug or a malicious
		// counterparty module. The bug may occur in bank or any part of the code that allows
		// the escrow address to be drained. A malicious counterparty module could drain the
		// escrow address by allowing more tokens to be sent back then were escrowed.
		return sdkerrors.Wrap(err, "unable to transfer escrow tokens, this may be caused by a malicious counterparty module or a bug: please open an issue on counterparty module")
	}

	return nil
}
