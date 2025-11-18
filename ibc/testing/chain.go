package ibctesting

import (
	"fmt"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcgotesting "github.com/cosmos/ibc-go/v10/testing"

	"github.com/haqq-network/haqq/app"
)

// ChainIDPrefix defines the default chain ID prefix for Haqq Network test chains
var ChainIDPrefix = "haqq_"

func init() {
	ibcgotesting.ChainIDPrefix = ChainIDPrefix
}

func CommitBlock(chain *ibcgotesting.TestChain, res *abci.ResponseFinalizeBlock) {
	// In ibc-go v10, commitBlock logic is internal, so we duplicate it here
	_, err := chain.App.Commit()
	require.NoError(chain.TB, err)

	// set the last header to the current header
	// use nil trusted fields
	chain.LatestCommittedHeader = chain.CurrentTMClientHeader()
	// set the trusted validator set to the next validator set
	chain.TrustedValidators[uint64(chain.ProposedHeader.Height)] = chain.NextVals

	// val set changes returned from previous block get applied to the next validators
	// of this block. See tendermint spec for details.
	chain.Vals = chain.NextVals
	chain.NextVals = ibcgotesting.ApplyValSetChanges(chain, chain.Vals, res.ValidatorUpdates)

	// increment the proposer priority of validators
	chain.Vals.IncrementProposerPriority(1)

	// increment the current header
	chain.ProposedHeader = cmtproto.Header{
		ChainID: chain.ChainID,
		Height:  chain.App.LastBlockHeight() + 1,
		AppHash: chain.App.LastCommitID().Hash,
		// NOTE: the time is increased by the coordinator to maintain time synchrony amongst
		// chains.
		Time:               chain.ProposedHeader.Time,
		ValidatorsHash:     chain.Vals.Hash(),
		NextValidatorsHash: chain.NextVals.Hash(),
		ProposerAddress:    chain.Vals.Proposer.Address,
	}
}

// SendMsgs delivers a transaction through the application. It updates the senders sequence
// number and updates the TestChain's headers. It returns the result and error if one
// occurred.
func SendMsgs(chain *ibcgotesting.TestChain, feeAmt int64, msgs ...sdk.Msg) (*abci.ExecTxResult, error) {
	var (
		bondDenom string
		err       error
	)

	if chain.SendMsgsOverride != nil {
		return chain.SendMsgsOverride(msgs...)
	}

	// ensure the chain has the latest time
	chain.Coordinator.UpdateTimeForChain(chain)

	if haqqChain, ok := chain.App.(*app.Haqq); ok {
		bondDenom, err = haqqChain.StakingKeeper.BondDenom(chain.GetContext())
	} else {
		bondDenom, err = chain.GetSimApp().StakingKeeper.BondDenom(chain.GetContext())
	}
	if err != nil {
		return nil, err
	}

	// increment acc sequence regardless of success or failure tx execution
	defer func() {
		err := chain.SenderAccount.SetSequence(chain.SenderAccount.GetSequence() + 1)
		if err != nil {
			panic(err)
		}
	}()

	fee := sdk.Coins{sdk.NewInt64Coin(bondDenom, feeAmt)}
	resp, err := SignAndDeliver(
		chain.TB,
		chain.GetContext(),
		chain.TxConfig,
		chain.App.GetBaseApp(),
		msgs,
		fee,
		chain.ChainID,
		[]uint64{chain.SenderAccount.GetAccountNumber()},
		[]uint64{chain.SenderAccount.GetSequence()},
		true,
		chain.ProposedHeader.GetTime(),
		chain.NextVals.Hash(),
		chain.SenderPrivKey,
	)
	if err != nil {
		return nil, err
	}

	// CommitBlock calls FinalizeBlock and Commit and apply the validator set changes
	CommitBlock(chain, resp)

	require.Len(chain.TB, resp.TxResults, 1)
	txResult := resp.TxResults[0]

	if txResult.Code != 0 {
		return txResult, fmt.Errorf("%s/%d: %q", txResult.Codespace, txResult.Code, txResult.Log)
	}

	chain.Coordinator.IncrementTime()

	return txResult, nil
}
