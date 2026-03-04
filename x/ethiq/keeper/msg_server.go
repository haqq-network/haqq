package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

var _ types.MsgServer = &msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the ethiq MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) MintHaqq(goCtx context.Context, msg *types.MsgMintHaqq) (*types.MsgMintHaqqResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsModuleEnabled(ctx) {
		return nil, types.ErrModuleDisabled
	}

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Addresses are already validated above
	toAddress := sdk.MustAccAddressFromBech32(msg.ToAddress)
	fromAddress := sdk.MustAccAddressFromBech32(msg.FromAddress)

	mintedHaqqAmt, err := k.Keeper.BurnIslmForHaqq(ctx, msg.IslmAmount, fromAddress, toAddress)
	if err != nil {
		return nil, err
	}

	return &types.MsgMintHaqqResponse{
		HaqqAmount: mintedHaqqAmt,
	}, nil
}

func (k msgServer) MintHaqqByApplication(goCtx context.Context, msg *types.MsgMintHaqqByApplication) (*types.MsgMintHaqqByApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsModuleEnabled(ctx) {
		return nil, types.ErrModuleDisabled
	}

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	burnApplication, err := types.GetApplicationByID(msg.ApplicationId)
	if err != nil {
		return nil, err
	}

	mintedHaqqAmt, err := k.Keeper.BurnIslmForHaqqByApplicationID(ctx, burnApplication.Id)
	if err != nil {
		return nil, err
	}

	return &types.MsgMintHaqqByApplicationResponse{
		HaqqAmount: mintedHaqqAmt,
		ToAddress:  burnApplication.ToAddress,
	}, nil
}
