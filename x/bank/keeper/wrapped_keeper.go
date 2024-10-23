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
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

type ERC20Keeper interface {
	IsERC20Enabled(ctx sdk.Context) bool
	GetTokenPairID(ctx sdk.Context, token string) []byte
	GetTokenPair(ctx sdk.Context, id []byte) (types.TokenPair, bool)
	GetTokenPairs(ctx sdk.Context) []types.TokenPair
	IterateTokenPairs(ctx sdk.Context, cb func(tokenPair types.TokenPair) (stop bool))
	BalanceOf(ctx sdk.Context, abi abi.ABI, contract, account common.Address) *big.Int
	ConvertCoin(goCtx context.Context, msg *types.MsgConvertCoin) (*types.MsgConvertCoinResponse, error)
	CallEVM(ctx sdk.Context, abi abi.ABI, from, contract common.Address, commit bool, method string, args ...interface{}) (*evmtypes.MsgEthereumTxResponse, error)
}

type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
	HasAccount(ctx sdk.Context, addr sdk.AccAddress) bool
	SetAccount(ctx sdk.Context, acc authtypes.AccountI)
	NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
}

type WrappedBaseKeeper struct {
	bankkeeper.Keeper
	ek      ERC20Keeper
	ak      AccountKeeper
	decoder sdk.TxDecoder
}

func NewWrappedBaseKeeper(
	bk bankkeeper.Keeper,
	ek ERC20Keeper,
	ak AccountKeeper,
	decoder sdk.TxDecoder,
) WrappedBaseKeeper {
	return WrappedBaseKeeper{
		Keeper:  bk,
		ek:      ek,
		ak:      ak,
		decoder: decoder,
	}
}

func (k WrappedBaseKeeper) UnwrapKeeper() bankkeeper.Keeper {
	return k.Keeper
}

func (k WrappedBaseKeeper) UnwrapBaseKeeper() bankkeeper.BaseKeeper {
	return k.Keeper.(bankkeeper.BaseKeeper)
}
