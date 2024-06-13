package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/haqq-network/haqq/x/shariahoracle/client/cli"
)

var (
	MintCACProposalHandler           = govclient.NewProposalHandler(cli.NewMintCACProposalCmd)
	BurnCACProposalHandler           = govclient.NewProposalHandler(cli.NewBurnCACProposalCmd)
	UpdateCACContractProposalHandler = govclient.NewProposalHandler(cli.NewUpdateCACContractProposalCmd)
)
