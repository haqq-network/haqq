package grpc

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/haqq-network/haqq/testutil/integration/common/network"
)

// Handler is an interface that defines the common methods that are used to query
// the network's modules via gRPC.
type Handler interface {
	// Account methods
	GetAccount(address string) (authtypes.AccountI, error)

	// Authz methods
	GetAuthorizations(grantee, granter string) ([]authz.Authorization, error)
	GetAuthorizationsByGrantee(grantee string) ([]authz.Authorization, error)
	GetAuthorizationsByGranter(granter string) ([]authz.Authorization, error)
	GetGrants(grantee, granter string) ([]*authz.Grant, error)
	GetGrantsByGrantee(grantee string) ([]*authz.GrantAuthorization, error)
	GetGrantsByGranter(granter string) ([]*authz.GrantAuthorization, error)

	// Bank methods
	GetBalance(address sdktypes.AccAddress, denom string) (*banktypes.QueryBalanceResponse, error)
	GetAllBalances(address sdktypes.AccAddress) (*banktypes.QueryAllBalancesResponse, error)

	// Staking methods
	GetDelegation(delegatorAddress string, validatorAddress string) (*stakingtypes.QueryDelegationResponse, error)
	GetBondedValidators() (*stakingtypes.QueryValidatorsResponse, error)
}

var _ Handler = (*IntegrationHandler)(nil)

// IntegrationHandler is a helper struct to query the network's modules
// via gRPC. This is to simulate the behavior of a real user and avoid querying
// the modules directly.
type IntegrationHandler struct {
	network network.Network
}

// NewIntegrationHandler creates a new IntegrationHandler instance.
func NewIntegrationHandler(network network.Network) *IntegrationHandler {
	return &IntegrationHandler{
		network: network,
	}
}
