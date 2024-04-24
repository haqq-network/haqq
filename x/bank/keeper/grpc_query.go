package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/contracts"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
)

var _ types.QueryServer = BaseKeeper{}

// Balance implements the Query/Balance gRPC method
func (k BaseKeeper) Balance(ctx context.Context, req *types.QueryBalanceRequest) (*types.QueryBalanceResponse, error) {
	res, err := k.BaseKeeper.Balance(ctx, req)
	if err != nil {
		return nil, err
	}

	ercBalance, err := k.getErc20BalanceByDenom(ctx, req.Address, req.Denom)
	if err != nil {
		return nil, err
	}

	if ercBalance.IsValid() {
		coin := res.Balance.Add(ercBalance)
		res.Balance = &coin
	}

	return res, nil
}

// AllBalances implements the Query/AllBalances gRPC method
func (k BaseKeeper) AllBalances(ctx context.Context, req *types.QueryAllBalancesRequest) (*types.QueryAllBalancesResponse, error) {
	res, err := k.BaseKeeper.AllBalances(ctx, req)
	if err != nil {
		return nil, err
	}

	ercBalances := k.getErc20Balances(ctx, req.Address)
	res.Balances = res.Balances.Sort().Add(ercBalances.Sort()...)

	return res, nil
}

// SpendableBalances implements a gRPC query handler for retrieving an account's
// spendable balances.
func (k BaseKeeper) SpendableBalances(ctx context.Context, req *types.QuerySpendableBalancesRequest) (*types.QuerySpendableBalancesResponse, error) {
	res, err := k.BaseKeeper.SpendableBalances(ctx, req)
	if err != nil {
		return nil, err
	}

	ercBalances := k.getErc20Balances(ctx, req.Address)
	res.Balances = res.Balances.Sort().Add(ercBalances.Sort()...)

	return res, nil
}

func (k BaseKeeper) SpendableBalanceByDenom(ctx context.Context, req *types.QuerySpendableBalanceByDenomRequest) (*types.QuerySpendableBalanceByDenomResponse, error) {
	res, err := k.BaseKeeper.SpendableBalanceByDenom(ctx, req)
	if err != nil {
		return nil, err
	}

	ercBalance, err := k.getErc20BalanceByDenom(ctx, req.Address, req.Denom)
	if err != nil {
		return nil, err
	}

	if ercBalance.IsValid() {
		coin := res.Balance.Add(ercBalance)
		res.Balance = &coin
	}

	return res, nil
}

func (k BaseKeeper) getErc20BalanceByDenom(ctx context.Context, addr, denom string) (sdk.Coin, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !k.ek.IsERC20Enabled(sdkCtx) {
		return sdk.Coin{}, nil
	}

	tokenPairID := k.ek.GetTokenPairID(sdkCtx, denom)
	if len(tokenPairID) == 0 {
		return sdk.Coin{}, nil
	}

	tokenPair, found := k.ek.GetTokenPair(sdkCtx, tokenPairID)
	if !found || !tokenPair.Enabled {
		return sdk.Coin{}, nil
	}

	// AccAddressFromBech32 error check already handled above in original method
	address, _ := sdk.AccAddressFromBech32(addr)
	evmAddr := common.BytesToAddress(address.Bytes())
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	contract := tokenPair.GetERC20Contract()
	balanceToken := k.ek.BalanceOf(sdkCtx, erc20, contract, evmAddr)
	if balanceToken == nil {
		return sdk.Coin{}, errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve balance")
	}

	return sdk.NewCoin(tokenPair.Denom, sdkmath.NewIntFromBigInt(balanceToken)), nil
}

func (k BaseKeeper) getErc20Balances(ctx context.Context, addr string) sdk.Coins {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !k.ek.IsERC20Enabled(sdkCtx) {
		return sdk.NewCoins()
	}

	// AccAddressFromBech32 error check already handled above in original method
	address, _ := sdk.AccAddressFromBech32(addr)
	evmAddr := common.BytesToAddress(address.Bytes())
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	resCoins := sdk.NewCoins()

	k.ek.IterateTokenPairs(sdkCtx, func(tokenPair erc20types.TokenPair) (stop bool) {
		if tokenPair.Enabled {
			contract := tokenPair.GetERC20Contract()
			balanceToken := k.ek.BalanceOf(sdkCtx, erc20, contract, evmAddr)
			if balanceToken == nil {
				// TODO Log error
				return false
			}

			resCoins = resCoins.Add(sdk.NewCoin(tokenPair.Denom, sdkmath.NewIntFromBigInt(balanceToken)))
		}

		return false
	})

	return resCoins
}
