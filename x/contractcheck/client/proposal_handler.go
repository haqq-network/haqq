package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/haqq-network/haqq/x/contractcheck/client/cli"
)

var MintNFTProposalHandler = govclient.NewProposalHandler(cli.NewMintNFTProposalCmd)
