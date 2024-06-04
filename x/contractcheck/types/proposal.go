package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	govcdc "github.com/cosmos/cosmos-sdk/x/gov/codec"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/ethereum/go-ethereum/common"
)

// constants
const (
	ProposalTypeMintNFT string = "MintNFT"
)

// Implements Proposal Interface
var (
	_ v1beta1.Content = &MintNFTProposal{}
)

func init() {
	v1beta1.RegisterProposalType(ProposalTypeMintNFT)
	govcdc.ModuleCdc.Amino.RegisterConcrete(&MintNFTProposal{}, "contractcheck/MintNFTProposal", nil)
}

// NewRegisterCoinProposal returns new instance of RegisterCoinProposal
func NewMintNFTProposal(title, description, contractAddress, destination, nftURL string) v1beta1.Content {
	return &MintNFTProposal{
		Title:              title,
		Description:        description,
		ContractAddress:    contractAddress,
		NftUrl:             nftURL,
		DestinationAddress: destination,
	}
}

// ProposalRoute returns router key for this proposal
func (*MintNFTProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns proposal type for this proposal
func (*MintNFTProposal) ProposalType() string {
	return ProposalTypeMintNFT
}

// ValidateBasic performs a stateless check of the proposal fields
func (rtbp *MintNFTProposal) ValidateBasic() error {
	if len(strings.TrimSpace(rtbp.NftUrl)) == 0 {
		return errorsmod.Wrap(types.ErrInvalidProposalContent, "nft url cannot be blank")
	}
	if !common.IsHexAddress(rtbp.ContractAddress) {
		return errorsmod.Wrap(types.ErrInvalidProposalContent, "invalid contract address")
	}
	if !common.IsHexAddress(rtbp.DestinationAddress) {
		return errorsmod.Wrap(types.ErrInvalidProposalContent, "invalid destination address")
	}
	return v1beta1.ValidateAbstract(rtbp)
}
