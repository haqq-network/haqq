package keeper

import (
	"context"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	sdkmath "cosmossdk.io/math"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ucdao/types"
)

var _ Keeper = (*BaseKeeper)(nil)

// Keeper defines a module interface that facilitates the transfer of coins
// between accounts.
type Keeper interface {
	GetTotalBalance(ctx sdk.Context) sdk.Coins
	GetPaginatedTotalBalance(ctx sdk.Context, pagination *query.PageRequest) (sdk.Coins, *query.PageResponse, error)
	IterateTotalBalance(ctx sdk.Context, cb func(sdk.Coin) bool)
	GetTotalBalanceOf(ctx sdk.Context, denom string) sdk.Coin
	HasTotalBalanceOf(ctx sdk.Context, denom string) bool

	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	HasBalance(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin) bool
	GetAccountBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	GetAccountsBalances(ctx sdk.Context) []types.Balance
	GetHolders(ctx sdk.Context) []sdk.AccAddress
	IsHolder(ctx sdk.Context, addr sdk.AccAddress) bool

	Fund(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error
	TransferOwnership(ctx sdk.Context, owner, newOwner sdk.AccAddress, amount sdk.Coins) (sdk.Coins, error)
	ConvertToEthiq(ctx sdk.Context, sender, receiver sdk.AccAddress, ethiqAmount, maxISLMAmount sdkmath.Int) (sdk.Coin, error)

	// grpc query endpoints
	Balance(ctx context.Context, req *types.QueryBalanceRequest) (*types.QueryBalanceResponse, error)
	AllBalances(ctx context.Context, req *types.QueryAllBalancesRequest) (*types.QueryAllBalancesResponse, error)
	TotalBalance(ctx context.Context, req *types.QueryTotalBalanceRequest) (*types.QueryTotalBalanceResponse, error)
	Holders(ctx context.Context, req *types.QueryHoldersRequest) (*types.QueryHoldersResponse, error)
	Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error)
	EscrowAddress(ctx context.Context, req *types.QueryEscrowAddressRequest) (*types.QueryEscrowAddressResponse, error)

	// genesis methods
	InitGenesis(ctx sdk.Context, genState *types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}

// BaseKeeper manages transfers between accounts. It implements the Keeper interface.
type BaseKeeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	ak     types.AccountKeeper
	bk     types.BankKeeper
	lvk    types.LiquidVestingKeeper
	ethiqk types.EthiqKeeper
}

func NewBaseKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	lvk types.LiquidVestingKeeper,
	ethiqk types.EthiqKeeper,
	authority string,
) BaseKeeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Errorf("invalid dao authority address: %w", err))
	}

	// TODO Add authority address to the keeper
	return BaseKeeper{
		cdc:      cdc,
		storeKey: storeKey,
		ak:       ak,
		bk:       bk,
		lvk:      lvk,
		ethiqk:   ethiqk,
	}
}

// Fund allows an account to directly fund the community fund pool.
// The amount is first added to the distribution module account and then directly
// added to the pool. An error is returned if the amount cannot be sent to the
// module account.
func (k BaseKeeper) Fund(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error {
	if !k.IsModuleEnabled(ctx) {
		return types.ErrModuleDisabled
	}

	escrowAddr := types.GetEscrowAddress(sender)
	for _, coin := range amount {
		if coin.Denom != utils.BaseDenom && !types.IsLiquidToken(coin.Denom) {
			return sdkerrors.Wrapf(types.ErrInvalidDenom, "denom %s is not allowed", coin.Denom)
		}

		if coin.IsZero() {
			continue
		}

		if err := k.escrowToken(ctx, sender, escrowAddr, coin); err != nil {
			return err
		}
	}

	// Update holders index
	k.setHoldersIndex(ctx, sender)

	return nil
}

func (k BaseKeeper) TransferOwnership(ctx sdk.Context, owner, newOwner sdk.AccAddress, amount sdk.Coins) (sdk.Coins, error) {
	if !k.IsModuleEnabled(ctx) {
		return nil, types.ErrModuleDisabled
	}

	balances := k.GetAccountBalances(ctx, owner)
	if balances.IsZero() {
		return nil, types.ErrNotEligible
	}

	ownerEscrowAddr := types.GetEscrowAddress(owner)
	newOwnerEscrowAddr := types.GetEscrowAddress(newOwner)

	for _, coin := range amount {
		if coin.IsZero() {
			// should not happen
			continue
		}

		if err := k.transferEscrowToken(ctx, ownerEscrowAddr, newOwnerEscrowAddr, coin); err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to transfer ownership of %s", coin)
		}
	}

	// Update holders index
	k.setHoldersIndex(ctx, newOwner)
	k.setHoldersIndex(ctx, owner)

	return amount, nil
}

func (k BaseKeeper) redeemISLM(ctx sdk.Context, sender sdk.AccAddress, redeemCoin sdk.Coin) (sdk.Coin, error) {
	balance := k.bk.GetBalance(ctx, sender, redeemCoin.Denom)
	if balance.IsZero() {
		return sdk.Coin{}, sdkerrors.Wrapf(types.ErrInsufficientFunds, "sender %s does not have %s balance", sender, redeemCoin.Denom)
	}

	redeemAmount := redeemCoin.Amount
	if balance.Amount.LT(redeemCoin.Amount) {
		redeemAmount = balance.Amount
	}

	err := k.lvk.Redeem(ctx, sender, sender, sdk.NewCoin(redeemCoin.Denom, redeemAmount))
	if err != nil {
		return sdk.Coin{}, sdkerrors.Wrapf(err, "failed to redeem ISLM from liquid vesting module")
	}

	return sdk.NewCoin(utils.BaseDenom, redeemAmount), nil
}

func (k BaseKeeper) trackISLMBalance(ctx sdk.Context, sender sdk.AccAddress, redeemedCoin sdk.Coin) {
	// redeemISLM function is sending redeemed ISLM to sender's escrow address directly, so we need to add it to total balance of aISLM in ucdao module
	// Update total balance of aISLM in ucdao module
	currentTotalISLMBalance := k.GetTotalBalanceOf(ctx, redeemedCoin.Denom)
	newTotalISLMBalance := currentTotalISLMBalance.Add(redeemedCoin)
	k.setTotalBalanceOfCoin(ctx, newTotalISLMBalance)

	// Update holders index
	k.setHoldersIndex(ctx, sender)
}

// ConvertToEthiq converts ISLM tokens to ethiq tokens for a holder.
func (k BaseKeeper) ConvertToEthiq(ctx sdk.Context, sender, receiver sdk.AccAddress, ethiqAmount, maxISLMAmount sdkmath.Int) (sdk.Coin, error) {
	if !k.IsModuleEnabled(ctx) {
		return sdk.Coin{}, types.ErrModuleDisabled
	}

	// Validation: user should be listed as one of the holders in ucdao module
	if !k.IsHolder(ctx, sender) {
		return sdk.Coin{}, types.ErrNotEligible
	}

	// Calculate required ISLM amount
	requiredISLM, _, err := k.ethiqk.CalculateRequiredISLM(ctx, ethiqAmount)
	if err != nil {
		return sdk.Coin{}, sdkerrors.Wrapf(err, "failed to calculate required ISLM")
	}

	// Return error if required ISLM amount is greater than max ISLM amount
	if requiredISLM.GT(maxISLMAmount) {
		return sdk.Coin{}, sdkerrors.Wrapf(types.ErrInsufficientFunds, "required ISLM %s is greater than max_ISLM_amount %s", requiredISLM, maxISLMAmount)
	}

	// Get sender's all balances
	senderBalances := k.GetAccountBalances(ctx, sender)
	senderBalancesAmount := sdkmath.ZeroInt()
	for _, balance := range senderBalances {
		senderBalancesAmount = senderBalancesAmount.Add(balance.Amount)
	}

	// Return error if sender's total balance is less than required ISLM amount
	if senderBalancesAmount.LT(requiredISLM) {
		return sdk.Coin{}, sdkerrors.Wrapf(types.ErrInsufficientFunds, "sender's total balance %s is less than required ISLM %s to convert to ethiq", senderBalancesAmount, requiredISLM)
	}

	senderISLMBalance := senderBalances.AmountOf(utils.BaseDenom)

	// If sender's ISLM balance is less than required ISLM amount, redeem from liquid vesting module
	if senderISLMBalance.LT(requiredISLM) {
		// Calculate the amount of ISLM to redeem from liquid vesting module
		requiredRedeemAmount := requiredISLM.Sub(senderISLMBalance)

		redeemedAmount := sdkmath.ZeroInt()

		// redeem non-ISLM balances from liquid vesting module until required ISLM amount is reached
		for _, balance := range senderBalances {
			if balance.Denom == utils.BaseDenom {
				continue
			}

			redeemCoin := sdk.NewCoin(balance.Denom, requiredRedeemAmount.Sub(redeemedAmount))

			// redeem balance from liquid vesting module
			redeemedCoin, err := k.redeemISLM(ctx, types.GetEscrowAddress(sender), redeemCoin)
			if err != nil {
				continue
			}

			// track ISLM Balance
			k.trackISLMBalance(ctx, sender, redeemedCoin)

			// add redeemed amount to total redeemed amount
			redeemedAmount = redeemedAmount.Add(redeemedCoin.Amount)
			// if total redeemed amount is greater than or equal to required ISLM amount, break
			if redeemedAmount.GTE(requiredRedeemAmount) {
				break
			}
		}

		// Return error if total redeemed amount is less than required redeem amount
		if redeemedAmount.LT(requiredRedeemAmount) {
			return sdk.Coin{}, sdkerrors.Wrapf(types.ErrInsufficientFunds, "sender's total redeemed amount %s is less than required redeem amount %s to convert to ethiq", redeemedAmount, requiredRedeemAmount)
		}
	}

	// Exchange aISLM to ethiq
	spentISLMAmount, err := k.ethiqk.ConvertToEthiq(ctx, ethiqAmount, maxISLMAmount, types.GetEscrowAddress(sender), receiver)
	if err != nil {
		return sdk.Coin{}, sdkerrors.Wrapf(err, "failed to convert amount of aISLM to ethiq")
	}

	// Update total balance of aISLM in ucdao module
	currentTotalISLMBalance := k.GetTotalBalanceOf(ctx, utils.BaseDenom)

	// Deduct exchanged ISLM amount from total balance of aISLM in ucdao module
	newTotalISLMBalance := currentTotalISLMBalance.Sub(sdk.NewCoin(utils.BaseDenom, spentISLMAmount))
	k.setTotalBalanceOfCoin(ctx, newTotalISLMBalance)

	// Update holders index
	k.setHoldersIndex(ctx, sender)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeConvertToEthiqExecuted,
			sdk.NewAttribute(types.AttributeKeySender, sender.String()),
			sdk.NewAttribute(types.AttributeKeyReceiver, receiver.String()),
			sdk.NewAttribute(types.AttributeKeyIslmSpent, spentISLMAmount.String()),
			sdk.NewAttribute(types.AttributeKeyEthiqAmount, ethiqAmount.String()),
		),
	)

	return sdk.NewCoin(utils.BaseDenom, spentISLMAmount), nil
}

// Logger returns a module-specific logger.
func (k BaseKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}
