package grpc

import (
	"context"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// GetStakingParams returns the EVM module params.
func (gqh *IntegrationHandler) GetStakingParams() (*stakingtypes.QueryParamsResponse, error) {
	stakingClinet := gqh.network.GetStakingClient()
	return stakingClinet.Params(context.Background(), &stakingtypes.QueryParamsRequest{})
}
