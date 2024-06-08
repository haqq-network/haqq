package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// NewQueryBalanceRequest creates a new instance of QueryBalanceRequest.
//
//nolint:interfacer
func NewQueryBalanceRequest(addr sdk.AccAddress, denom string) *QueryBalanceRequest {
	return &QueryBalanceRequest{Address: addr.String(), Denom: denom}
}

// NewQueryAllBalancesRequest creates a new instance of QueryAllBalancesRequest.
//
//nolint:interfacer
func NewQueryAllBalancesRequest(addr sdk.AccAddress, req *query.PageRequest) *QueryAllBalancesRequest {
	return &QueryAllBalancesRequest{Address: addr.String(), Pagination: req}
}
