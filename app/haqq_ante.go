package app

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
)

var ErrCommunitySpendingComingLater = errorsmod.Register("haqq-ante", 6001, "community pool spend coming later")

//nolint:all
func NewHaqqAnteHandlerDecorator(_ keeper.Keeper, h types.AnteHandler) types.AnteHandler {
	return func(ctx types.Context, tx types.Tx, simulate bool) (newCtx types.Context, err error) {
		msgs := tx.GetMsgs()

		for i := 0; i < len(msgs); i++ {
			isValid := true

			switch msg := msgs[i].(type) {
			case *govv1beta1.MsgSubmitProposal:
				if strings.HasSuffix(msg.Content.TypeUrl, "CommunityPoolSpendProposal") {
					isValid = false
				}
			case *govv1.MsgExecLegacyContent:
				if strings.HasSuffix(msg.Content.TypeUrl, "CommunityPoolSpendProposal") {
					isValid = false
				}
			case *govv1.MsgSubmitProposal:
				proposalMsgs, err := msg.GetMsgs()
				if err != nil {
					return ctx, errorsmod.Wrap(err, "proposal contains invalid message(s)")
				}
				for _, proposalMsg := range proposalMsgs {
					if _, ok := proposalMsg.(*distrtypes.MsgCommunityPoolSpend); ok {
						isValid = false
						break
					}
				}
			}

			if !isValid {
				return ctx, ErrCommunitySpendingComingLater
			}
		}

		return h(ctx, tx, simulate)
	}
}
