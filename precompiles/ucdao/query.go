package ucdao

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// Allowance returns the remaining allowance of a grantee to the contract.
func (p Precompile) Allowance(
	ctx sdk.Context,
	method *abi.Method,
	_ *vm.Contract,
	args []interface{},
) ([]byte, error) {
	grantee, granter, msg, err := authorization.CheckAllowanceArgs(args)
	if err != nil {
		return nil, err
	}

	msgAuthz, _ := p.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), msg)

	if msgAuthz == nil {
		return method.Outputs.Pack(big.NewInt(0))
	}

	// A type switch, not two assertions: the previous shape kept the value of the
	// *first* assertion and read SpendLimit off it after the *second* one had
	// accepted the grant, so a TransferOwnershipAuthorization dereferenced a nil
	// *ConvertToHaqqAuthorization. allowance is a view method reachable from
	// eth_call, and cmn.HandleGasError re-raises anything that is not
	// ErrorOutOfGas, so that panic left the precompile.
	switch grant := msgAuthz.(type) {
	case *ucdaotypes.ConvertToHaqqAuthorization:
		if grant.SpendLimit == nil {
			return method.Outputs.Pack(abi.MaxUint256)
		}
		return method.Outputs.Pack(grant.SpendLimit.Amount.BigInt())
	case *ucdaotypes.TransferOwnershipAuthorization:
		if grant.SpendLimit == nil {
			return method.Outputs.Pack(abi.MaxUint256)
		}
		return method.Outputs.Pack(grant.SpendLimit.Amount.BigInt())
	default:
		return nil, fmt.Errorf(cmn.ErrInvalidType, "ucdao authorization", fmt.Sprintf("%T or %T", &ucdaotypes.ConvertToHaqqAuthorization{}, &ucdaotypes.TransferOwnershipAuthorization{}), msgAuthz)
	}
}
