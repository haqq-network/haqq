package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/ucdao/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the distribution MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) Fund(goCtx context.Context, msg *types.MsgFund) (*types.MsgFundResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// already validated in ValidateBasic
	addr := sdk.MustAccAddressFromBech32(msg.Depositor)

	if err := k.Keeper.Fund(ctx, msg.Amount, addr); err != nil {
		return nil, err
	}

	return &types.MsgFundResponse{}, nil
}

func (k msgServer) TransferOwnership(goCtx context.Context, msg *types.MsgTransferOwnership) (*types.MsgTransferOwnershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// already validated in ValidateBasic
	owner := sdk.MustAccAddressFromBech32(msg.Owner)
	newOwner := sdk.MustAccAddressFromBech32(msg.NewOwner)

	// Transfer all balances
	balances := k.GetAccountBalances(ctx, owner)
	if balances.IsZero() {
		return nil, types.ErrNotEligible
	}

	if _, err := k.Keeper.TransferOwnership(ctx, owner, newOwner, balances); err != nil {
		return nil, err
	}

	return &types.MsgTransferOwnershipResponse{}, nil
}

func (k msgServer) TransferOwnershipWithRatio(goCtx context.Context, msg *types.MsgTransferOwnershipWithRatio) (*types.MsgTransferOwnershipWithRatioResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// already validated in ValidateBasic
	owner := sdk.MustAccAddressFromBech32(msg.Owner)
	newOwner := sdk.MustAccAddressFromBech32(msg.NewOwner)

	balances := k.GetAccountBalances(ctx, owner)
	if balances.IsZero() {
		return nil, types.ErrNotEligible
	}

	coins := sdk.NewCoins()
	for _, coin := range balances {
		amt := coin.Amount.ToLegacyDec().Mul(msg.Ratio).TruncateInt()
		coins = append(coins, sdk.NewCoin(coin.Denom, amt))
	}

	transferred, err := k.Keeper.TransferOwnership(ctx, owner, newOwner, coins)
	if err != nil {
		return nil, err
	}

	return &types.MsgTransferOwnershipWithRatioResponse{
		Coins: transferred,
	}, nil
}

func (k msgServer) TransferOwnershipWithAmount(goCtx context.Context, msg *types.MsgTransferOwnershipWithAmount) (*types.MsgTransferOwnershipWithAmountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// already validated in ValidateBasic
	owner := sdk.MustAccAddressFromBech32(msg.Owner)
	newOwner := sdk.MustAccAddressFromBech32(msg.NewOwner)

	balances := k.GetAccountBalances(ctx, owner)
	if balances.IsZero() {
		return nil, types.ErrNotEligible
	}

	if _, err := k.Keeper.TransferOwnership(ctx, owner, newOwner, msg.Amount); err != nil {
		return nil, err
	}

	return &types.MsgTransferOwnershipWithAmountResponse{}, nil
}
