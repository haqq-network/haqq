package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/contracts"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
)

var _ types.QueryServer = BaseKeeper{}

// Balance implements the Query/Balance gRPC method
func (k WrappedBaseKeeper) Balance(ctx context.Context, req *types.QueryBalanceRequest) (*types.QueryBalanceResponse, error) {
	res, err := k.wbk.Balance(ctx, req)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !k.ek.IsERC20Enabled(sdkCtx) {
		return res, nil
	}

	// AccAddressFromBech32 error check already handled above in original method
	address, _ := sdk.AccAddressFromBech32(req.Address)
	tokenPairs := k.ek.GetTokenPairs(sdkCtx)
	evmAddr := common.BytesToAddress(address.Bytes())
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI

	for _, tokenPair := range tokenPairs {
		if !tokenPair.Enabled {
			continue
		}

		if tokenPair.Denom != req.Denom {
			continue
		}

		contract := tokenPair.GetERC20Contract()
		balanceToken := k.ek.BalanceOf(sdkCtx, erc20, contract, evmAddr)
		if balanceToken == nil {
			return nil, errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve balance")
		}
		balanceTokenCoin := sdk.NewCoin(tokenPair.Denom, sdk.NewIntFromBigInt(balanceToken))

		coin := res.Balance.Add(balanceTokenCoin)
		res.Balance = &coin
	}

	return res, nil
}

// AllBalances implements the Query/AllBalances gRPC method
func (k WrappedBaseKeeper) AllBalances(ctx context.Context, req *types.QueryAllBalancesRequest) (*types.QueryAllBalancesResponse, error) {
	res, err := k.wbk.AllBalances(ctx, req)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !k.ek.IsERC20Enabled(sdkCtx) {
		return res, nil
	}

	// AccAddressFromBech32 error check already handled above in original method
	address, _ := sdk.AccAddressFromBech32(req.Address)
	tokenPairs := k.ek.GetTokenPairs(sdkCtx)
	evmAddr := common.BytesToAddress(address.Bytes())
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI

	for _, tokenPair := range tokenPairs {
		if !tokenPair.Enabled {
			continue
		}

		contract := tokenPair.GetERC20Contract()
		balanceToken := k.ek.BalanceOf(sdkCtx, erc20, contract, evmAddr)
		if balanceToken == nil {
			return nil, errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve balance")
		}
		balanceTokenCoin := sdk.NewCoin(tokenPair.Denom, sdk.NewIntFromBigInt(balanceToken))

		res.Balances = res.Balances.Add(balanceTokenCoin)
	}

	return res, nil
}

// SpendableBalances implements a gRPC query handler for retrieving an account's
// spendable balances.
func (k WrappedBaseKeeper) SpendableBalances(ctx context.Context, req *types.QuerySpendableBalancesRequest) (*types.QuerySpendableBalancesResponse, error) {
	res, err := k.wbk.SpendableBalances(ctx, req)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !k.ek.IsERC20Enabled(sdkCtx) {
		return res, nil
	}

	// AccAddressFromBech32 error check already handled above in original method
	address, _ := sdk.AccAddressFromBech32(req.Address)
	tokenPairs := k.ek.GetTokenPairs(sdkCtx)
	evmAddr := common.BytesToAddress(address.Bytes())
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI

	for _, tokenPair := range tokenPairs {
		if !tokenPair.Enabled {
			continue
		}

		contract := tokenPair.GetERC20Contract()
		balanceToken := k.ek.BalanceOf(sdkCtx, erc20, contract, evmAddr)
		if balanceToken == nil {
			return nil, errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve balance")
		}
		balanceTokenCoin := sdk.NewCoin(tokenPair.Denom, sdk.NewIntFromBigInt(balanceToken))

		res.Balances = res.Balances.Add(balanceTokenCoin)
	}

	return res, nil
}

// TotalSupply implements the Query/TotalSupply gRPC method
func (k WrappedBaseKeeper) TotalSupply(ctx context.Context, req *types.QueryTotalSupplyRequest) (*types.QueryTotalSupplyResponse, error) {
	return k.wbk.TotalSupply(ctx, req)
}

// SupplyOf implements the Query/SupplyOf gRPC method
func (k WrappedBaseKeeper) SupplyOf(c context.Context, req *types.QuerySupplyOfRequest) (*types.QuerySupplyOfResponse, error) {
	return k.wbk.SupplyOf(c, req)
}

// Params implements the gRPC service handler for querying x/bank parameters.
func (k WrappedBaseKeeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return k.wbk.Params(ctx, req)
}

// DenomsMetadata implements Query/DenomsMetadata gRPC method.
func (k WrappedBaseKeeper) DenomsMetadata(c context.Context, req *types.QueryDenomsMetadataRequest) (*types.QueryDenomsMetadataResponse, error) {
	return k.wbk.DenomsMetadata(c, req)
}

// DenomMetadata implements Query/DenomMetadata gRPC method.
func (k WrappedBaseKeeper) DenomMetadata(c context.Context, req *types.QueryDenomMetadataRequest) (*types.QueryDenomMetadataResponse, error) {
	return k.wbk.DenomMetadata(c, req)
}

func (k WrappedBaseKeeper) DenomOwners(
	goCtx context.Context,
	req *types.QueryDenomOwnersRequest,
) (*types.QueryDenomOwnersResponse, error) {
	return k.wbk.DenomOwners(goCtx, req)
}