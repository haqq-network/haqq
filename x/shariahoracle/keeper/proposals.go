package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
)

// GrantCAC mints CAC
func (k Keeper) GrantCAC(ctx sdk.Context, to string) error {
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

// RevokeCAC burns CAC
func (k Keeper) RevokeCAC(ctx sdk.Context, from string) error {
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
		"upgradeToAndCall",
		common.HexToAddress(newContractAddress),
		[]byte{},
	)
	if err != nil {
		return err
	}

	return nil
}
