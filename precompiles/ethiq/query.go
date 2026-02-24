package ethiq

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

const (
	// Calculate defines the ABI query method name for the calculation of ethiq coins to be minted for a given aISLM.
	Calculate = "calculate"
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
		if strings.Contains(err.Error(), ErrCalculationFailed) {
			return method.Outputs.Pack(big.NewInt(0), big.NewInt(0), big.NewInt(0), "")
		}
		return nil, err
	}

	return method.Outputs.Pack(
		res.EstimatedHaqqAmount.BigInt(),
		res.SupplyBefore.BigInt(),
		res.SupplyAfter.BigInt(),
		res.AveragePrice.String(),
	)
}

// Allowance returns the remaining allowance of a grantee to the contract.
func (p Precompile) Allowance(
	ctx sdk.Context,
	method *abi.Method,
	_ *vm.Contract,
	args []interface{},
) ([]byte, error) {
	grantee, granter, msgTypeUrl, err := authorization.CheckAllowanceArgs(args)
	if err != nil {
		return nil, err
	}

	msgAuthz, _ := p.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), msgTypeUrl)

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

	return nil, fmt.Errorf(cmn.ErrInvalidType, "ethiq authorization", &ethiqtypes.MintHaqqAuthorization{}, msgAuthz)
}
