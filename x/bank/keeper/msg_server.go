package keeper

import (
	"context"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/haqq-network/haqq/contracts"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

type msgServer struct {
	WrappedBaseKeeper
}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper WrappedBaseKeeper) types.MsgServer {
	return &msgServer{WrappedBaseKeeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) Send(goCtx context.Context, msg *types.MsgSend) (*types.MsgSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.IsSendEnabledCoins(ctx, msg.Amount...); err != nil {
		return nil, err
	}

	from, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		return nil, err
	}
	to, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, err
	}

	if k.BlockedAddr(to) {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", msg.ToAddress)
	}

	if err := k.sendCoinsWithERC20(ctx, from, to, msg.Amount); err != nil {
		return nil, err
	}

	defer func() {
		for _, a := range msg.Amount {
			if a.Amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "send"},
					float32(a.Amount.Int64()),
					[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
				)
			}
		}
	}()

	return &types.MsgSendResponse{}, nil
}

// MultiSend copy of original method. TODO Add ERC20 support
func (k msgServer) MultiSend(goCtx context.Context, msg *types.MsgMultiSend) (*types.MsgMultiSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// NOTE: totalIn == totalOut should already have been checked
	for _, in := range msg.Inputs {
		if err := k.IsSendEnabledCoins(ctx, in.Coins...); err != nil {
			return nil, err
		}
	}

	for _, out := range msg.Outputs {
		accAddr := sdk.MustAccAddressFromBech32(out.Address)

		if k.BlockedAddr(accAddr) {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive transactions", out.Address)
		}
	}

	err := k.InputOutputCoins(ctx, msg.Inputs, msg.Outputs)
	if err != nil {
		return nil, err
	}

	return &types.MsgMultiSendResponse{}, nil
}

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, sdkerrors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) SetSendEnabled(goCtx context.Context, msg *types.MsgSetSendEnabled) (*types.MsgSetSendEnabledResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, sdkerrors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if len(msg.SendEnabled) > 0 {
		k.SetAllSendEnabled(ctx, msg.SendEnabled)
	}
	if len(msg.UseDefaultFor) > 0 {
		k.DeleteSendEnabled(ctx, msg.UseDefaultFor...)
	}

	return &types.MsgSetSendEnabledResponse{}, nil
}

func (k msgServer) sendCoinsWithERC20(ctx sdk.Context, from sdk.AccAddress, to sdk.AccAddress, amt sdk.Coins) error {
	// Use original SendCoins method is ERC20 is disabled
	if !k.ek.IsERC20Enabled(ctx) {
		return k.SendCoins(ctx, from, to, amt)
	}

	nativeCoins := sdk.NewCoins()
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Convert and transfer registered ERC20 tokens
	for _, coin := range amt {
		tokenPairID := k.ek.GetTokenPairID(sdkCtx, coin.Denom)
		if len(tokenPairID) == 0 {
			// If no such pair registered for the given denom, try to send as a native coin
			nativeCoins = append(nativeCoins, coin)
			continue
		}

		tokenPair, found := k.ek.GetTokenPair(sdkCtx, tokenPairID)
		if !found || !tokenPair.Enabled {
			// if tokenPair is Disabled or not found, try to transfer on Cosmos layer without conversion
			nativeCoins = append(nativeCoins, coin)
			continue
		}

		if err := k.subUnlockedERC20Tokens(ctx, tokenPair, from, to, coin); err != nil {
			return err
		}

		// Create account if recipient does not exist.
		//
		// NOTE: This should ultimately be removed in favor a more flexible approach
		// such as delegated fee messages.
		accExists := k.ak.HasAccount(ctx, to)
		if !accExists {
			defer telemetry.IncrCounter(1, "new", "account")
			k.ak.SetAccount(ctx, k.ak.NewAccountWithAddress(ctx, to))
		}
	}

	// Send native the rest native coins
	return k.SendCoins(ctx, from, to, nativeCoins)
}

func (k msgServer) subUnlockedERC20Tokens(ctx sdk.Context, tokenPair erc20types.TokenPair, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coin) error {
	if !amt.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, amt.String())
	}

	lockedCoins := k.LockedCoins(ctx, fromAddr)

	balance := k.GetBalance(ctx, fromAddr, amt.Denom)
	locked := sdk.NewCoin(amt.Denom, lockedCoins.AmountOf(amt.Denom))
	spendable := balance.Sub(locked)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := tokenPair.GetERC20Contract()
	evmFromAddr := common.BytesToAddress(fromAddr.Bytes())
	evmFromBalanceToken := k.ek.BalanceOf(sdkCtx, erc20, contract, evmFromAddr)
	if evmFromBalanceToken == nil {
		return errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve sender's balance")
	}
	evmBalance := sdk.NewCoin(tokenPair.Denom, sdk.NewIntFromBigInt(evmFromBalanceToken))

	_, hasNeg := sdk.Coins{spendable.Add(evmBalance)}.SafeSub(amt)
	if hasNeg {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, "%s is smaller than %s", spendable, amt)
	}

	if !spendable.IsZero() {
		// Build MsgConvertCoin, from recipient to recipient since IBC transfer already occurred
		msg := erc20types.NewMsgConvertCoin(spendable, evmFromAddr, fromAddr)

		if _, err := k.ek.ConvertCoin(sdk.WrapSDKContext(ctx), msg); err != nil {
			return errorsmod.Wrap(err, "failed to convert coins")
		}
	}

	evmToAddr := common.BytesToAddress(toAddr.Bytes())
	evmToBalanceTokenBefore := k.ek.BalanceOf(sdkCtx, erc20, contract, evmToAddr)
	if evmToBalanceTokenBefore == nil {
		return errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve receiver's balance")
	}
	// Transfer Tokens to receiver
	res, err := k.evm.CallEVM(ctx, erc20, evmFromAddr, contract, true, "transfer", evmToAddr, amt.Amount.BigInt())
	if err != nil {
		return errorsmod.Wrap(err, "failed to transfer erc20 tokens: call evm")
	}

	// Check unpackedRet execution
	var unpackedRet erc20types.ERC20BoolResponse
	if err := erc20.UnpackIntoInterface(&unpackedRet, "transfer", res.Ret); err != nil {
		return errorsmod.Wrap(err, "failed to transfer erc20 tokens: unpack")
	}

	if !unpackedRet.Value {
		return errorsmod.Wrap(err, "failed to transfer erc20 tokens")
	}

	// Check expected balance after transfer execution
	evmToBalanceTokenAfter := k.ek.BalanceOf(sdkCtx, erc20, contract, evmToAddr)
	if evmToBalanceTokenAfter == nil {
		return errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve receiver's balance")
	}

	expToken := big.NewInt(0).Add(evmToBalanceTokenBefore, amt.Amount.BigInt())

	if r := evmToBalanceTokenAfter.Cmp(expToken); r != 0 {
		return errorsmod.Wrapf(
			erc20types.ErrBalanceInvariance,
			"invalid token balance - expected: %v, actual: %v",
			expToken, evmToBalanceTokenAfter,
		)
	}

	// Check for unexpected `Approval` event in logs
	if err := k.monitorApprovalEvent(res); err != nil {
		return err
	}

	// No events to be sent, all necessary events have been emitted by ConvertCoin method, other logic is on EVM layer

	return nil
}

// monitorApprovalEvent returns an error if the given transactions logs include
// an unexpected `Approval` event
// NOTE: copy from ERC20Keeper
func (k msgServer) monitorApprovalEvent(res *evmtypes.MsgEthereumTxResponse) error {
	if res == nil || len(res.Logs) == 0 {
		return nil
	}

	logApprovalSig := []byte("Approval(address,address,uint256)")
	logApprovalSigHash := crypto.Keccak256Hash(logApprovalSig)

	for _, log := range res.Logs {
		if log.Topics[0] == logApprovalSigHash.Hex() {
			return errorsmod.Wrapf(
				erc20types.ErrUnexpectedEvent, "unexpected Approval event",
			)
		}
	}

	return nil
}
