package app

import (
	"errors"
	"strings"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var ErrCommunitySpendingComingLater = sdkerrors.Register("haqq-ante", 6001, "community fund spend coming later")

func NewHaqqAnteHandlerDecorator(sk keeper.Keeper, h types.AnteHandler) types.AnteHandler {
	return func(ctx types.Context, tx types.Tx, simulate bool) (newCtx types.Context, err error) {
		msgs := tx.GetMsgs()

		for i := 0; i < len(msgs); i++ {
			isValid := true

			switch msgs[i].(type) {
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

			if !isValid {
				return ctx, errors.New("tx cannot be executed")
			}
		}

		return h(ctx, tx, simulate)
	}
}
