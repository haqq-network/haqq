package keeper

import (
	"context"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
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
	ConvertToHaqq(ctx sdk.Context, sender, receiver sdk.AccAddress, islmAmount sdkmath.Int) (sdk.Coin, error)

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

	// export methods to use them from ethiq module
	TrackAddBalance(sdk.Context, sdk.Coin)
	TrackSubBalance(sdk.Context, sdk.Coin)
	SetHoldersIndex(sdk.Context, sdk.AccAddress)
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
	k.SetHoldersIndex(ctx, sender)

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

	if err := k.transferEscrowToken(ctx, ownerEscrowAddr, newOwnerEscrowAddr, amount); err != nil {
		return nil, sdkerrors.Wrap(err, "failed to transfer ownership of coins")
	}

	// Update holders index
	k.SetHoldersIndex(ctx, newOwner)
	k.SetHoldersIndex(ctx, owner)

	return amount, nil
}

// ConvertToHaqq converts ISLM tokens to ethiq tokens for a holder.
func (k BaseKeeper) ConvertToHaqq(ctx sdk.Context, sender, receiver sdk.AccAddress, islmAmount sdkmath.Int) (sdk.Coin, error) {
	if !k.IsModuleEnabled(ctx) {
		return sdk.Coin{}, types.ErrModuleDisabled
	}

	// Validation: user should be listed as one of the holders in ucdao module
	if !k.IsHolder(ctx, sender) {
		return sdk.Coin{}, types.ErrNotEligible
	}

	// Get sender's all balances
	senderBalances := k.GetAccountBalances(ctx, sender)
	senderBalancesAmount := sdkmath.ZeroInt()
	for _, balance := range senderBalances {
		senderBalancesAmount = senderBalancesAmount.Add(balance.Amount)
	}

	// Return error if sender's total balance is less than required ISLM amount
	if senderBalancesAmount.LT(islmAmount) {
		return sdk.Coin{}, sdkerrors.Wrapf(types.ErrInsufficientFunds, "sender's total balance is less than required amount: %s < %s", senderBalancesAmount, islmAmount)
	}

	senderEscrowAddr := types.GetEscrowAddress(sender)

	// redeem all aLIQUID balances from liquid vesting module
	for _, balance := range senderBalances {
		if balance.Denom == utils.BaseDenom {
			continue
		}

		// redeem balance from liquid vesting module
		if err := k.lvk.Redeem(ctx, senderEscrowAddr, senderEscrowAddr, balance); err != nil {
			return sdk.Coin{}, sdkerrors.Wrap(err, "failed to redeem liquid coins")
		}

		// track internal module total balances
		// add redeemed aISLM
		k.TrackAddBalance(ctx, sdk.NewCoin(utils.BaseDenom, balance.Amount))
		// sub redeemed aLIQUID
		k.TrackSubBalance(ctx, balance)
	}

	mintedHaqqAmt, err := k.ethiqk.BurnIslmForHaqq(ctx, islmAmount, senderEscrowAddr, receiver)
	if err != nil {
		return sdk.Coin{}, sdkerrors.Wrap(err, "failed to convert amount of aISLM to aHAQQ")
	}

	// Update total balance of aISLM in ucdao module
	k.TrackSubBalance(ctx, sdk.NewCoin(utils.BaseDenom, islmAmount))

	// Update holders index
	k.SetHoldersIndex(ctx, sender)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeConvertToEthiqExecuted,
			sdk.NewAttribute(types.AttributeKeySender, sender.String()),
			sdk.NewAttribute(types.AttributeKeyReceiver, receiver.String()),
			sdk.NewAttribute(types.AttributeKeyIslmSpent, islmAmount.String()),
			sdk.NewAttribute(types.AttributeKeyEthiqAmount, mintedHaqqAmt.String()),
		),
	)

	return sdk.NewCoin(ethiqtypes.BaseDenom, mintedHaqqAmt), nil
}

// Logger returns a module-specific logger.
func (k BaseKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}
