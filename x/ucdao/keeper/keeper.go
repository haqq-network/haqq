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
	ConvertToEthiq(ctx sdk.Context, sender, receiver sdk.AccAddress, ethiqAmount, maxISLMAmount sdkmath.Int) error

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

// ConvertToEthiq converts ISLM tokens to ethiq tokens for a holder.
func (k BaseKeeper) ConvertToEthiq(ctx sdk.Context, sender, receiver sdk.AccAddress, ethiqAmount, maxISLMAmount sdkmath.Int) error {
	if !k.IsModuleEnabled(ctx) {
		return types.ErrModuleDisabled
	}

	// Get spender's escrow address
	escrowAddr := types.GetEscrowAddress(sender)

	// Validation: user should be listed as one of the holders in ucdao module
	if !k.IsHolder(ctx, escrowAddr) {
		return types.ErrNotEligible
	}

	// Get spender's ISLM balance
	spenderISLMBalance := k.GetBalance(ctx, sender, utils.BaseDenom)
	if spenderISLMBalance.Amount.LT(maxISLMAmount) {
		// calculate the necessary amount of ISLM to redeem from liquid vesting module
		necessaryISLMAmount := maxISLMAmount.Sub(spenderISLMBalance.Amount)

		// Should proceed redeem from liquid vesting module
		err := k.lvk.Redeem(ctx, sender, sdk.NewCoin(utils.BaseDenom, necessaryISLMAmount))
		if err != nil {
			return sdkerrors.Wrapf(err, "failed to redeem ISLM from liquid vesting module")
		}

		// Should fund redeemed aISLM to ucdao module escrow address
		err = k.Fund(ctx, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, necessaryISLMAmount)), sender)
		if err != nil {
			return sdkerrors.Wrapf(err, "failed to fund redeemed ISLM to ucdao module escrow address")
		}
	}

	requiredISLMAmount, err := k.ethiqk.ConvertToEthiq(ctx, ethiqAmount, maxISLMAmount, sender, receiver)
	if err != nil {
		return sdkerrors.Wrapf(err, "failed to convert amount of aISLM to ethiq")
	}

	// Update total balance tracking for ISLM
	currentTotalEscrow := k.GetTotalBalanceOf(ctx, utils.BaseDenom)

	// should remove required ISLAM amount instead of maxISLM amount
	newTotalEscrow := currentTotalEscrow.Sub(sdk.NewCoin(utils.BaseDenom, requiredISLMAmount))
	k.setTotalBalanceOfCoin(ctx, newTotalEscrow)

	// Update holders index
	k.setHoldersIndex(ctx, sender)

	return nil
}

// Logger returns a module-specific logger.
func (k BaseKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}
