package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the liquidvesting MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// Liquidate liquidates specified amount of token locked in vesting into liquid token
func (k msgServer) Liquidate(goCtx context.Context, msg *types.MsgLiquidate) (*types.MsgLiquidateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// already validated in ValidateBasic
	liquidateFromAddress := sdk.MustAccAddressFromBech32(msg.LiquidateFrom)
	liquidateToAddress := liquidateFromAddress
	if msg.LiquidateTo != msg.LiquidateFrom {
		liquidateToAddress = sdk.MustAccAddressFromBech32(msg.LiquidateTo)
	}

	liquidTokenCoin, contractAddr, err := k.Keeper.Liquidate(ctx, liquidateFromAddress, liquidateToAddress, msg.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgLiquidateResponse{
		Minted:       liquidTokenCoin,
		ContractAddr: contractAddr,
	}, nil
}

// Redeem redeems specified amount of liquid token into original locked token and adds them to account
func (k msgServer) Redeem(goCtx context.Context, msg *types.MsgRedeem) (*types.MsgRedeemResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// already validated in ValidateBasic
	fromAddress := sdk.MustAccAddressFromBech32(msg.RedeemFrom)
	toAddress := sdk.MustAccAddressFromBech32(msg.RedeemTo)

	if err := k.Keeper.Redeem(ctx, fromAddress, toAddress, msg.Amount); err != nil {
		return nil, err
	}

	return &types.MsgRedeemResponse{}, nil
}
