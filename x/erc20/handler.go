package erc20

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	erc20types "github.com/evmos/evmos/v14/x/erc20/types"
)

// NewHandler defines the erc20 module handler instance
func NewHandler(server erc20types.MsgServer) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *erc20types.MsgConvertCoin:
			res, err := server.ConvertCoin(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *erc20types.MsgConvertERC20:
			res, err := server.ConvertERC20(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *erc20types.MsgUpdateParams:
			res, err := server.UpdateParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			err := errorsmod.Wrapf(errortypes.ErrUnknownRequest, "unrecognized %s message type: %T", erc20types.ModuleName, msg)
			return nil, err
		}
	}
}
