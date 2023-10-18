package v162

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/haqq-network/haqq/x/vesting/types"
)

func fixVestingAccounts(ctx sdk.Context, ak authkeeper.AccountKeeper) error {
	logger := ctx.Logger()
	logger.Info("Updating Vesting accounts...")

	for _, addr := range accounts {
		accAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return err
		}

		acc := ak.GetAccount(ctx, accAddr)
		if acc == nil {
			return errorsmod.Wrapf(sdkerrors.ErrUnknownAddress, addr)
		}

		vestingAcc, ok := acc.(*types.ClawbackVestingAccount)
		if !ok {
			return fmt.Errorf("account %s is not a vesting account", addr)
		}

		codeHash := common.BytesToHash(crypto.Keccak256(nil))
		vestingAcc.CodeHash = codeHash.Hex()
		ak.SetAccount(ctx, vestingAcc)
	}

	logger.Info("Vesting accounts successfully updated")
	return nil
}
