package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/haqq-network/haqq/x/shariahoracle/client/cli"
)

var (
	GrantCACProposalHandler          = govclient.NewProposalHandler(cli.NewGrantCACProposalCmd)
	RevokeCACProposalHandler         = govclient.NewProposalHandler(cli.NewRevokeCACProposalCmd)
	UpdateCACContractProposalHandler = govclient.NewProposalHandler(cli.NewUpdateCACContractProposalCmd)
)
