package app

import (
	"errors"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/haqq-network/haqq/utils"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var (
	ErrCommunitySpendingComingLater = sdkerrors.Register("haqq-ante", 6001, "community fund spend coming later")
	ErrVestingComingLater           = sdkerrors.Register("haqq-ante", 6002, "vesting coming later")
)

func NewHaqqAnteHandlerDecorator(_ keeper.Keeper, h sdk.AnteHandler) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		msgs := tx.GetMsgs()

		for i := 0; i < len(msgs); i++ {
			isValid := true

			switch msg := msgs[i].(type) {
			case *vestingtypes.MsgConvertIntoVestingAccount, *vestingtypes.MsgCreateClawbackVestingAccount:
				sigTx, ok := tx.(authsigning.SigVerifiableTx)
				if !ok {
					return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid tx type")
				}

				signers := sigTx.GetSigners()
				for j := 0; j < len(signers); j++ {
					if !utils.IsAllowedVestingFunderAccount(signers[j].String()) {
						isValid = false
						break
					}
				}

				if !isValid {
					return ctx, ErrVestingComingLater
				}
			case *govtypes.MsgSubmitProposal:
				disabledProposals := []string{"CommunityPoolSpendProposal"}

				for _, ap := range disabledProposals {
					if strings.HasSuffix(msg.Content.TypeUrl, ap) {
						isValid = false
					}
				}

				if !isValid {
					return ctx, ErrCommunitySpendingComingLater
				}
			}

			if !isValid {
				return ctx, errors.New("tx cannot be executed")
			}
		}

		return h(ctx, tx, simulate)
	}
}
