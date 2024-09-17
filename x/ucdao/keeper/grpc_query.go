package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/haqq-network/haqq/x/ucdao/types"
)

var _ types.QueryServer = BaseKeeper{}

// Balance implements the Query/Balance gRPC method
func (k BaseKeeper) Balance(ctx context.Context, req *types.QueryBalanceRequest) (*types.QueryBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := sdk.ValidateDenom(req.Denom); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var (
		address []byte
		err     error
	)

	if common.IsHexAddress(req.Address) {
		hexAddr := common.HexToAddress(req.Address)
		address = hexAddr.Bytes()
	} else {
		address, err = sdk.AccAddressFromBech32(req.Address)
	}
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid address: %s", err.Error())
	}

	balance := k.GetBalance(sdkCtx, address, req.Denom)

	return &types.QueryBalanceResponse{Balance: &balance}, nil
}

// AllBalances implements the Query/AllBalances gRPC method
func (k BaseKeeper) AllBalances(ctx context.Context, req *types.QueryAllBalancesRequest) (*types.QueryAllBalancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	var (
		address []byte
		err     error
	)

	if common.IsHexAddress(req.Address) {
		hexAddr := common.HexToAddress(req.Address)
		address = hexAddr.Bytes()
	} else {
		address, err = sdk.AccAddressFromBech32(req.Address)
	}
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid address: %s", err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	balances := sdk.NewCoins()
	accountStore := k.getAccountStore(sdkCtx, address)

	pageRes, err := query.Paginate(accountStore, req.Pagination, func(key, value []byte) error {
		denom := string(key)
		balance, err := UnmarshalBalanceCompat(k.cdc, value, denom)
		if err != nil {
			return err
		}
		balances = append(balances, balance)
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}

	return &types.QueryAllBalancesResponse{Balances: balances, Pagination: pageRes}, nil
}

// TotalBalance implements the Query/TotalSupply gRPC method
func (k BaseKeeper) TotalBalance(ctx context.Context, req *types.QueryTotalBalanceRequest) (*types.QueryTotalBalanceResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	totalBalance, pageRes, err := k.GetPaginatedTotalBalance(sdkCtx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTotalBalanceResponse{TotalBalance: totalBalance, Pagination: pageRes}, nil
}

// Params implements the gRPC service handler for querying x/bank parameters.
func (k BaseKeeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := k.GetParams(sdkCtx)

	return &types.QueryParamsResponse{Params: params}, nil
}
