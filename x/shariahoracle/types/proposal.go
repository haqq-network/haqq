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
	ProposalTypeMintCAC           string = "MintCAC"
	ProposalTypeBurnCAC           string = "BurnCAC"
	ProposalTypeUpdateCACContract string = "UpdateCACContract"
)

// Implements Proposal Interface
var (
	_ v1beta1.Content = &MintCACProposal{}
	_ v1beta1.Content = &BurnCACProposal{}
	_ v1beta1.Content = &UpdateCACContractProposal{}
)

func init() {
	v1beta1.RegisterProposalType(ProposalTypeMintCAC)
	v1beta1.RegisterProposalType(ProposalTypeBurnCAC)
	v1beta1.RegisterProposalType(ProposalTypeUpdateCACContract)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&MintCACProposal{}, "shariahoracle/MintCACProposal", nil)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&BurnCACProposal{}, "shariahoracle/BurnCACProposal", nil)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&UpdateCACContractProposal{}, "shariahoracle/UpdateCACContractProposal", nil)
}

// NewRegisterCoinProposal returns new instance of RegisterCoinProposal
func NewMintCACProposal(title, description string, granteeAddress ...string) v1beta1.Content {
	return &MintCACProposal{
		Title:       title,
		Description: description,
		Grantees:    granteeAddress,
	}
}

// ProposalRoute returns router key for this proposal
func (*MintCACProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*MintCACProposal) ProposalType() string {
	return ProposalTypeMintCAC
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *MintCACProposal) ValidateBasic() error {
	for _, grantee := range rtbp.Grantees {
		if !common.IsHexAddress(grantee) {
			return errorsmod.Wrap(types.ErrInvalidProposalContent, "invalid contract address")
		}
	}

	return v1beta1.ValidateAbstract(rtbp)
}

// NewBurnCACProposal returns new instance of RegisterCoinProposal
func NewBurnCACProposal(title, description string, granteeAddress ...string) v1beta1.Content {
	return &BurnCACProposal{
		Title:       title,
		Description: description,
		Grantees:    granteeAddress,
	}
}

// ProposalRoute returns router key for this proposal
func (*BurnCACProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*BurnCACProposal) ProposalType() string {
	return ProposalTypeBurnCAC
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *BurnCACProposal) ValidateBasic() error {
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
