package grpc

import (
	"context"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// GetDelegation returns the delegation for the given delegator and validator addresses.
func (gqh *IntegrationHandler) GetDelegation(delegatorAddress string, validatorAddress string) (*stakingtypes.QueryDelegationResponse, error) {
	stakingClient := gqh.network.GetStakingClient()
	return stakingClient.Delegation(context.Background(), &stakingtypes.QueryDelegationRequest{
		DelegatorAddr: delegatorAddress,
		ValidatorAddr: validatorAddress,
	})
}

// GetValidators returns the list of all bonded validators.
func (gqh *IntegrationHandler) GetBondedValidators() (*stakingtypes.QueryValidatorsResponse, error) {
	stakingClient := gqh.network.GetStakingClient()
	return stakingClient.Validators(context.Background(), &stakingtypes.QueryValidatorsRequest{
		Status: stakingtypes.BondStatusBonded,
	})
}
