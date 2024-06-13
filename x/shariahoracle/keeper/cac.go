package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
)

// MintCAC mints CAC
func (k Keeper) MintCAC(ctx sdk.Context, to string) error {
	// mint CAC
	contract := common.HexToAddress(k.GetCACContractAddress(ctx))

	_, err := k.CallEVM(ctx,
		contracts.CommunityApprovalCertificatesContract.ABI,
		types.ModuleAddress,
		contract,
		true,
		"safeMint",
		common.HexToAddress(to),
	)
	if err != nil {
		return err
	}

	return nil
}

// BurnCAC burns CAC
func (k Keeper) BurnCAC(ctx sdk.Context, from string) error {
	// burn CAC
	contract := common.HexToAddress(k.GetCACContractAddress(ctx))

	_, err := k.CallEVM(ctx,
		contracts.CommunityApprovalCertificatesContract.ABI,
		types.ModuleAddress,
		contract,
		true,
		"burn",
		common.HexToAddress(from),
	)
	if err != nil {
		return err
	}

	return nil
}

// UpdateCACContract updates CAC contract
func (k Keeper) UpdateCACContract(ctx sdk.Context, newContractAddress string) error {
	// burn CAC
	contract := common.HexToAddress(k.GetCACContractAddress(ctx))

	_, err := k.CallEVM(ctx,
		contracts.CommunityApprovalCertificatesContract.ABI,
		types.ModuleAddress,
		contract,
		true,
		"upgradeTo",
		common.HexToAddress(newContractAddress),
	)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) DoesAddressHaveCAC(ctx sdk.Context, address string) (bool, error) {
	var (
		contract = common.HexToAddress(k.GetCACContractAddress(ctx))
		cac      = contracts.CommunityApprovalCertificatesContract.ABI
		account  = common.HexToAddress(address)
	)

	res, err := k.CallEVM(ctx, cac, types.ModuleAddress, contract, true, "balanceOf", account)
	if err != nil {
		return false, err
	}
	unpacked, err := cac.Unpack("balanceOf", res.Ret)
	if err != nil || len(unpacked) == 0 {
		return false, nil
	}

	balance, ok := unpacked[0].(*big.Int)
	if !ok {
		return false, nil
	}

	return balance.Cmp(big.NewInt(1)) == 0, nil
}
