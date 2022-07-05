package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var ErrDelegationComingLater = sdkerrors.Register("haqq-ante", 6000, "delegation coming later")
var ErrCommunitySpendingComingLater = sdkerrors.Register("haqq-ante", 6001, "community fund spend coming later")

func NewHaqqAnteHandlerDecorator(sk keeper.Keeper, h types.AnteHandler) types.AnteHandler {
	return func(ctx types.Context, tx types.Tx, simulate bool) (newCtx types.Context, err error) {
		msgs := tx.GetMsgs()

		for i := 0; i < len(msgs); i++ {
			isValid := true

			fmt.Printf("\n start ante msgs :%#v\n", msgs[i])

			switch msgs[i].(type) {
			case *stakingtypes.MsgDelegate, *stakingtypes.MsgCreateValidator:
				isValid = false

				if ctx.BlockHeight() == 0 {
					continue
				}

				sigTx, ok := tx.(authsigning.SigVerifiableTx)
				if !ok {
					return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid tx type")
				}

				signers := sigTx.GetSigners()
				validators := sk.GetAllValidators(ctx)

				for j := 0; j < len(signers); j++ {
					for k := 0; k < len(validators); k++ {
						if signers[j].Equals(validators[k].GetOperator()) {
							isValid = true
							break
						}
					}
				}

				if !isValid {
					return ctx, ErrDelegationComingLater
				}

			case *govtypes.MsgSubmitProposal:
				disabledProposals := []string{"CommunityPoolSpendProposal"}
				govMsg := msgs[i].(*govtypes.MsgSubmitProposal)

				for _, ap := range disabledProposals {
					if strings.HasSuffix(govMsg.Content.TypeUrl, ap) {
						isValid = false
					}
				}

				if !isValid {
					return ctx, ErrCommunitySpendingComingLater
				}
			}

			fmt.Printf("\n start ante msgs :%#v, isValid: %v\n", msgs, isValid)

			if !isValid {
				return ctx, errors.New("tx cannot be executed")
			}
		}

		return h(ctx, tx, simulate)
	}
}
