package types

import (
	errorsmod "cosmossdk.io/errors"
	govcdc "github.com/cosmos/cosmos-sdk/x/gov/codec"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/ethereum/go-ethereum/common"
)

// constants
const (
	ProposalTypeGrantCAC          string = "GrantCAC"
	ProposalTypeRevokeCAC         string = "RevokeCAC"
	ProposalTypeUpdateCACContract string = "UpdateCACContract"
)

// Implements Proposal Interface
var (
	_ v1beta1.Content = &GrantCACProposal{}
	_ v1beta1.Content = &RevokeCACProposal{}
	_ v1beta1.Content = &UpdateCACContractProposal{}
)

func init() {
	v1beta1.RegisterProposalType(ProposalTypeGrantCAC)
	v1beta1.RegisterProposalType(ProposalTypeRevokeCAC)
	v1beta1.RegisterProposalType(ProposalTypeUpdateCACContract)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&GrantCACProposal{}, "shariahoracle/GrantCACProposal", nil)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&RevokeCACProposal{}, "shariahoracle/RevokeCACProposal", nil)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&UpdateCACContractProposal{}, "shariahoracle/UpdateCACContractProposal", nil)
}

// NewRegisterCoinProposal returns new instance of RegisterCoinProposal
func NewGrantCACProposal(title, description string, granteeAddress ...string) v1beta1.Content {
	return &GrantCACProposal{
		Title:       title,
		Description: description,
		Grantees:    granteeAddress,
	}
}

// ProposalRoute returns router key for this proposal
func (*GrantCACProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*GrantCACProposal) ProposalType() string {
	return ProposalTypeGrantCAC
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *GrantCACProposal) ValidateBasic() error {
	for _, grantee := range rtbp.Grantees {
		if !common.IsHexAddress(grantee) {
			return errorsmod.Wrap(types.ErrInvalidProposalContent, "invalid contract address")
		}
	}

	return v1beta1.ValidateAbstract(rtbp)
}

// NewRevokeCACProposal returns new instance of RegisterCoinProposal
func NewRevokeCACProposal(title, description string, granteeAddress ...string) v1beta1.Content {
	return &RevokeCACProposal{
		Title:       title,
		Description: description,
		Grantees:    granteeAddress,
	}
}

// ProposalRoute returns router key for this proposal
func (*RevokeCACProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*RevokeCACProposal) ProposalType() string {
	return ProposalTypeRevokeCAC
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *RevokeCACProposal) ValidateBasic() error {
	for _, grantee := range rtbp.Grantees {
		if !common.IsHexAddress(grantee) {
			return errorsmod.Wrap(types.ErrInvalidProposalContent, "invalid contract address")
		}
	}

	return v1beta1.ValidateAbstract(rtbp)
}

// NewRegisterCoinProposal returns new instance of RegisterCoinProposal
func NewUpdateCACContractProposal(title, description, newImplementation string) v1beta1.Content {
	return &UpdateCACContractProposal{
		Title:                    title,
		Description:              description,
		NewImplementationAddress: newImplementation,
	}
}

// ProposalRoute returns router key for this proposal
func (*UpdateCACContractProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*UpdateCACContractProposal) ProposalType() string {
	return ProposalTypeUpdateCACContract
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *UpdateCACContractProposal) ValidateBasic() error {
	if !common.IsHexAddress(rtbp.NewImplementationAddress) {
		return errorsmod.Wrap(types.ErrInvalidProposalContent, "invalid contract address")
	}

	return v1beta1.ValidateAbstract(rtbp)
}
