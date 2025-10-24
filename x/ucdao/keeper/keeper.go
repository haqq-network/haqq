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
	IterateAccountBalances(ctx sdk.Context, addr sdk.AccAddress, cb func(sdk.Coin) bool)
	GetAccountBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	IterateAllBalances(ctx sdk.Context, cb func(sdk.AccAddress, sdk.Coin) bool)
	GetAccountsBalances(ctx sdk.Context) []types.Balance

	Fund(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error
	TransferOwnership(ctx sdk.Context, owner, newOwner sdk.AccAddress, amount sdk.Coins) (sdk.Coins, error)

	// grpc query endpoints
	Balance(ctx context.Context, req *types.QueryBalanceRequest) (*types.QueryBalanceResponse, error)
	AllBalances(ctx context.Context, req *types.QueryAllBalancesRequest) (*types.QueryAllBalancesResponse, error)
	TotalBalance(ctx context.Context, req *types.QueryTotalBalanceRequest) (*types.QueryTotalBalanceResponse, error)
	Holders(ctx context.Context, req *types.QueryHoldersRequest) (*types.QueryHoldersResponse, error)
	Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error)

	// genesis methods
	InitGenesis(ctx sdk.Context, genState *types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}

// BaseKeeper manages transfers between accounts. It implements the Keeper interface.
type BaseKeeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	ak types.AccountKeeper
	bk types.BankKeeper
}

func NewBaseKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak types.AccountKeeper,
	bk types.BankKeeper,
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

	if err := k.bk.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amount); err != nil {
		return err
	}

	for _, coin := range amount {
		if coin.Denom != utils.BaseDenom && !IsLiquidToken(coin.Denom) {
			return sdkerrors.Wrapf(types.ErrInvalidDenom, "denom %s is not allowed", coin.Denom)
		}

		if coin.IsZero() {
			continue
		}

		err := k.addCoinsToAccount(ctx, sender, sdk.NewCoins(coin))
		if err != nil {
			return err
		}

		bal := k.GetTotalBalanceOf(ctx, coin.Denom)
		bal = bal.Add(coin)
		k.setTotalBalanceOfCoin(ctx, bal)
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

	leftovers := sdk.NewCoins()
	for _, coin := range amount {
		if coin.IsZero() {
			// should not happen
			continue
		}

		ok, foundInBalance := balances.Find(coin.Denom)
		if !ok {
			return nil, sdkerrors.Wrapf(types.ErrInsufficientFunds, "zero balance of %s", coin.Denom)
		}
		leftCoin, err := foundInBalance.SafeSub(coin)
		if err != nil {
			return nil, sdkerrors.Wrapf(types.ErrInsufficientFunds, "%s on balance is lower than %s to transfer", foundInBalance, coin)
		}

		leftovers = append(leftovers, leftCoin)
	}

	// Add coins to new owner
	err := k.addCoinsToAccount(ctx, newOwner, amount)
	if err != nil {
		return nil, err
	}

	// Remove coins from old owner
	for _, coin := range leftovers {
		if err := k.setBalance(ctx, owner, coin); err != nil {
			return nil, err
		}
	}

	// Update holders index
	k.setHoldersIndex(ctx, newOwner)
	k.setHoldersIndex(ctx, owner)

	return amount, nil
}

// Logger returns a module-specific logger.
func (k BaseKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}
