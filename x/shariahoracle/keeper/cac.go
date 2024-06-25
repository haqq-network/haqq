package keeper

import (
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
)

// DoesAddressHaveCAC checks if an address has a Community Approval Certificate
func (k Keeper) DoesAddressHaveCAC(ctx sdk.Context, address string) (bool, error) {
	var (
		contract = common.HexToAddress(k.GetCACContractAddress(ctx))
		cac      = contracts.CommunityApprovalCertificatesContract.ABI
		account  = common.HexToAddress(address)
	)

	res, err := k.CallEVM(ctx, cac, types.ModuleAddress, contract, false, "balanceOf", account)
	if err != nil {
		return false, err
	}
	unpacked, err := cac.Unpack("balanceOf", res.Ret)
	if err != nil || len(unpacked) == 0 {
		return false, err
	}

	balance, ok := unpacked[0].(*big.Int)
	if !ok {
		return false, errors.Wrap(types.ErrInvalidEVMResponse, "failed to convert balance to *big.Int")
	}

	return balance.Cmp(big.NewInt(1)) == 0, nil
}
