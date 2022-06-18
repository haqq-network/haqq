package app

import (
	"github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var ErrDelegationComingLater = sdkerrors.Register("haqq-ante", 6000, "delegation coming later")

func NewHaqqAnteHandlerDecorator(sk keeper.Keeper, h types.AnteHandler) types.AnteHandler {
	return func(ctx types.Context, tx types.Tx, simulate bool) (newCtx types.Context, err error) {
		if ctx.BlockHeight() == 0 {
			return h(ctx, tx, simulate)
		}

		canPerform := true
		msgs := tx.GetMsgs()
		for i, ln := 0, len(msgs); canPerform && i < ln; i++ {
			switch msgs[i].(type) {
			case *stakingtypes.MsgDelegate, *stakingtypes.MsgCreateValidator:
				canPerform = false
			}
		}
		if canPerform {
			return h(ctx, tx, simulate)
		}

		sigTx, ok := tx.(authsigning.SigVerifiableTx)
		if !ok {
			return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid tx type")
		}

		signers := sigTx.GetSigners()
		validators := sk.GetAllValidators(ctx)
		canPerform = len(signers) > 0
		for i, sigLn := 0, len(signers); canPerform && i < sigLn; i++ {
			found := false
			for j, valLn := 0, len(validators); !found && j < valLn; j++ {
				found = signers[i].Equals(validators[j].GetOperator())
			}
			canPerform = found
		}
		if canPerform {
			return h(ctx, tx, simulate)
		}

		return ctx, ErrDelegationComingLater
	}
}
