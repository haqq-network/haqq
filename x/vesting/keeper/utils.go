package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/haqq-network/haqq/x/vesting/types"
)

// GetClawbackVestingAccount is a helper function to get the account from the
// account keeper and check if it is of the correct type for clawback vesting.
func (k Keeper) GetClawbackVestingAccount(goCtx context.Context, addr sdk.AccAddress) (*types.ClawbackVestingAccount, error) {
	acc := k.accountKeeper.GetAccount(goCtx, addr)
	if acc == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrUnknownAddress, "account at address '%s' does not exist", addr.String())
	}

	clawbackAccount, isClawback := acc.(*types.ClawbackVestingAccount)
	if !isClawback {
		return nil, errorsmod.Wrap(types.ErrNotSubjectToClawback, addr.String())
	}

	return clawbackAccount, nil
}
