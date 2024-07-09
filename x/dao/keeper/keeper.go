package keeper

import (
	"context"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/dao/types"
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
	IterateAllBalances(ctx sdk.Context, cb func(sdk.AccAddress, sdk.Coin) bool)
	GetAccountsBalances(ctx sdk.Context) []types.Balance

	Fund(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error

	// grpc query endpoints
	Balance(ctx context.Context, req *types.QueryBalanceRequest) (*types.QueryBalanceResponse, error)
	AllBalances(ctx context.Context, req *types.QueryAllBalancesRequest) (*types.QueryAllBalancesResponse, error)
	TotalBalance(ctx context.Context, req *types.QueryTotalBalanceRequest) (*types.QueryTotalBalanceResponse, error)
	AllVoicePower(goCtx context.Context, req *types.QueryAllVoicePowerRequest) (*types.QueryAllVoicePowerResponse, error)
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

	return nil
}

// Logger returns a module-specific logger.
func (k BaseKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}
