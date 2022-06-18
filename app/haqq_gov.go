package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var ErrFundGovComingLater = sdkerrors.Register("haqq-gov", 6000, "fund governance coming later")

func NewHaqqGovHandlerDecorator(routerType string, _ govtypes.Handler) govtypes.Handler {
	msg := fmt.Sprintf("%s rejected", routerType)
	return func(ctx sdk.Context, content govtypes.Content) error {
		ctx.Logger().
			With("type", content.ProposalType()).
			With("title", content.GetTitle()).
			Error(msg)
		return ErrFundGovComingLater
	}
}
