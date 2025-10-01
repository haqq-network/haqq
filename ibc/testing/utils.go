package ibctesting

import (
	"fmt"

	"github.com/stretchr/testify/require"

	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/haqq-network/haqq/app"
)

// VoteAndCheckProposalStatus votes on a gov proposal, checks if the proposal has passed, and returns an error if it has not with the failure reason.
func VoteAndCheckProposalStatus(endpoint *Endpoint, proposalID uint64) error {
	var (
		err    error
		params govtypesv1.Params
		p      govtypesv1.Proposal
	)
	// vote on proposal
	haqqApp, isHaqq := endpoint.Chain.App.(*app.Haqq)
	ctx := endpoint.Chain.GetContext()
	if isHaqq {
		require.NoError(endpoint.Chain.TB, haqqApp.GovKeeper.AddVote(ctx, proposalID, endpoint.Chain.SenderAccount.GetAddress(), govtypesv1.NewNonSplitVoteOption(govtypesv1.OptionYes), ""))

		params, err = haqqApp.GovKeeper.Params.Get(ctx)
	} else {
		require.NoError(endpoint.Chain.TB, endpoint.Chain.GetSimApp().GovKeeper.AddVote(ctx, proposalID, endpoint.Chain.SenderAccount.GetAddress(), govtypesv1.NewNonSplitVoteOption(govtypesv1.OptionYes), ""))

		params, err = endpoint.Chain.GetSimApp().GovKeeper.Params.Get(ctx)
	}
	require.NoError(endpoint.Chain.TB, err)

	// fast forward the chain context to end the voting period
	endpoint.Chain.Coordinator.IncrementTimeBy(*params.VotingPeriod + *params.MaxDepositPeriod)
	endpoint.Chain.NextBlock()

	// check if proposal passed or failed on msg execution
	// we need to grab the context again since the previous context is no longer valid as the chain header time has been incremented
	if isHaqq {
		p, err = haqqApp.GovKeeper.Proposals.Get(endpoint.Chain.GetContext(), proposalID)
		require.NoError(endpoint.Chain.TB, err)
	} else {
		p, err = endpoint.Chain.GetSimApp().GovKeeper.Proposals.Get(endpoint.Chain.GetContext(), proposalID)
		require.NoError(endpoint.Chain.TB, err)
	}
	if p.Status != govtypesv1.StatusPassed {
		return fmt.Errorf("proposal failed: %s", p.FailedReason)
	}
	return nil
}
