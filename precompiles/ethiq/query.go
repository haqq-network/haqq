package ethiq

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

const (
	// Calculate defines the ABI query method name for the calculation of ethiq coins to be minted for a given aISLM.
	Calculate = "calculate"
	// CalculateForApplication defines the ABI query method name for the calculation of ethiq coins
	// to be minted for a given application ID.
	CalculateForApplication = "calculateForApplication"
)

// Calculate returns the estimated amount of aHAQQ coins to be minted for a given aISLM.
func (p Precompile) Calculate(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	req, err := NewCalculateRequest(args)
	if err != nil {
		return nil, err
	}

	res, err := p.ethiqKeeper.Calculate(ctx, req)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(
		res.EstimatedHaqqAmount.BigInt(),
		res.SupplyBefore.BigInt(),
		res.SupplyAfter.BigInt(),
		res.AveragePrice.String(),
	)
}

// CalculateForApplication returns the estimated amount of aHAQQ coins to be minted for a given application ID.
func (p Precompile) CalculateForApplication(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	req, err := NewCalculateForApplicationRequest(args)
	if err != nil {
		return nil, err
	}

	res, err := p.ethiqKeeper.CalculateForApplication(ctx, req)
	if err != nil {
		return nil, err
	}

	receiver := common.BytesToAddress(sdk.MustAccAddressFromBech32(res.ToAddress).Bytes())

	return method.Outputs.Pack(
		res.EstimatedHaqqAmount.BigInt(),
		res.SupplyBefore.BigInt(),
		res.SupplyAfter.BigInt(),
		res.AveragePrice.String(),
		receiver,
	)
}

// Allowance returns the remaining allowance of a grantee to the contract.
//
// Only MintHaqqAuthorization has a scalar allowance. An application-based grant holds a list of
// approved application IDs, which does not fit the uint256 this method returns and must not be
// squeezed into it: the same return value would then carry an aISLM amount for one message type
// and a count for another, with nothing to tell them apart. Such a query is answered off chain
// through the standard authz Grants query, which decodes the authorization in full.
func (p Precompile) Allowance(
	ctx sdk.Context,
	method *abi.Method,
	_ *vm.Contract,
	args []interface{},
) ([]byte, error) {
	grantee, granter, msgTypeURL, err := authorization.CheckAllowanceArgs(args)
	if err != nil {
		return nil, err
	}

	msgAuthz, _ := p.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), msgTypeURL)

	if msgAuthz == nil {
		return method.Outputs.Pack(big.NewInt(0))
	}

	// Check for MintHaqqAuthorization
	if mintAuthz, ok := msgAuthz.(*ethiqtypes.MintHaqqAuthorization); ok {
		if mintAuthz.SpendLimit == nil {
			return method.Outputs.Pack(abi.MaxUint256)
		}
		return method.Outputs.Pack(mintAuthz.SpendLimit.Amount.BigInt())
	}

	return nil, fmt.Errorf(
		"allowance is only defined for %s; %s holds a list of application IDs, "+
			"query it with cosmos.authz.v1beta1.Query/Grants",
		MintHaqqMsgURL, msgTypeURL,
	)
}
