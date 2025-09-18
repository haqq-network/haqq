package grpc

import (
	"context"

	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

// GetRewardCoefficient returns current rewards coefficient.
func (gqh *IntegrationHandler) GetRewardCoefficient() (*coinomicstypes.QueryRewardCoefficientResponse, error) {
	coinomicsClient := gqh.network.GetCoinomicsClient()
	return coinomicsClient.RewardCoefficient(
		context.Background(),
		&coinomicstypes.QueryRewardCoefficientRequest{},
	)
}

// GetMaxSupply returns current value of max supply.
func (gqh *IntegrationHandler) GetMaxSupply() (*coinomicstypes.QueryMaxSupplyResponse, error) {
	coinomicsClient := gqh.network.GetCoinomicsClient()
	return coinomicsClient.MaxSupply(
		context.Background(),
		&coinomicstypes.QueryMaxSupplyRequest{},
	)
}

// GetParams returns current module parameters.
func (gqh *IntegrationHandler) GetParams() (*coinomicstypes.QueryParamsResponse, error) {
	coinomicsClient := gqh.network.GetCoinomicsClient()
	return coinomicsClient.Params(
		context.Background(),
		&coinomicstypes.QueryParamsRequest{},
	)
}
