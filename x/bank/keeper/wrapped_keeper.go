package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/x/erc20/types"
)

type ERC20Keeper interface {
	IsERC20Enabled(ctx sdk.Context) bool
	GetTokenPairID(ctx sdk.Context, token string) []byte
	GetTokenPair(ctx sdk.Context, id []byte) (types.TokenPair, bool)
	GetTokenPairs(ctx sdk.Context) []types.TokenPair
	IterateTokenPairs(ctx sdk.Context, cb func(tokenPair types.TokenPair) (stop bool))
	BalanceOf(ctx sdk.Context, abi abi.ABI, contract, account common.Address) *big.Int
	ConvertCoin(goCtx context.Context, msg *types.MsgConvertCoin) (*types.MsgConvertCoinResponse, error)
}

type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
	HasAccount(ctx sdk.Context, addr sdk.AccAddress) bool
	SetAccount(ctx sdk.Context, acc authtypes.AccountI)
	NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
}

type WrappedBaseKeeper struct {
	bankkeeper.Keeper
	evm types.EVMKeeper
	ek  ERC20Keeper
	ak  AccountKeeper
}

func NewWrappedBaseKeeper(
	bk bankkeeper.Keeper,
	evm types.EVMKeeper,
	ek ERC20Keeper,
	ak AccountKeeper,
) WrappedBaseKeeper {
	return WrappedBaseKeeper{
		Keeper: bk,
		evm:    evm,
		ek:     ek,
		ak:     ak,
	}
}

func (k WrappedBaseKeeper) UnwrapKeeper() bankkeeper.Keeper {
	return k.Keeper
}

func (k WrappedBaseKeeper) UnwrapBaseKeeper() bankkeeper.BaseKeeper {
	return k.Keeper.(bankkeeper.BaseKeeper)
}
