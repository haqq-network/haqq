package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/x/erc20/types"
)

type ERC20Keeper interface {
	IsERC20Enabled(ctx sdk.Context) bool
	GetTokenPairs(ctx sdk.Context) []types.TokenPair
	BalanceOf(ctx sdk.Context, abi abi.ABI, contract, account common.Address) *big.Int
}

type WrappedBaseKeeper struct {
	wbk bankkeeper.Keeper
	ek  ERC20Keeper
}

func NewWrappedBaseKeeper(
	bk bankkeeper.Keeper,
	ek ERC20Keeper,
) WrappedBaseKeeper {
	return WrappedBaseKeeper{
		wbk: bk,
		ek:  ek,
	}
}

func (k WrappedBaseKeeper) UnwrapBaseKeeper() bankkeeper.BaseKeeper {
	return k.wbk.(bankkeeper.BaseKeeper)
}
