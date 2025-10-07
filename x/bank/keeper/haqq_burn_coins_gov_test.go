package keeper_test

import (
	"fmt"
	"testing"
	"time"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/testutil/integration/haqq/utils"
	haqqutils "github.com/haqq-network/haqq/utils"
)

type BurnCoinsTestSuite struct {
	suite.Suite

	network     *network.UnitTestNetwork
	handler     grpc.Handler
	keyring     keyring.Keyring
	keyringVals keyring.Keyring
	factory     factory.TxFactory
}

func TestBurnCoinsTestSuite(t *testing.T) {
	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "HAQQ Bank Burn Coins Tests")
}

func (suite *BurnCoinsTestSuite) SetupTest() {
	keys := keyring.New(2)
	keysVals := keyring.New(4)

	sdk.DefaultBondDenom = haqqutils.BaseDenom
	fiveSec, err := time.ParseDuration("5s")
	Expect(err).To(BeNil())
	customGenesis := network.CustomGenesisState{}
	govGenesis := govv1.DefaultGenesisState()
	govGenesis.Params.VotingPeriod = &fiveSec
	govGenesis.Params.MaxDepositPeriod = &fiveSec
	customGenesis[govtypes.ModuleName] = govGenesis

	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
		network.WithAmountOfValidators(4),
		network.WithValidatorOperators(keysVals.GetAllAccAddrs()),
		network.WithCustomGenesis(customGenesis),
	)
	gh := grpc.NewIntegrationHandler(nw)
	tf := factory.New(nw, gh)

	suite.network = nw
	suite.factory = tf
	suite.handler = gh
	suite.keyring = keys
	suite.keyringVals = keysVals
}

var _ = Describe("Performing Burn tests", Ordered, func() {
	var (
		s            *BurnCoinsTestSuite
		authority    = authtypes.NewModuleAddress(govtypes.ModuleName).String()
		govParams    *govv1.Params
		cpInitialBal sdk.DecCoins
		proposal     govv1.Proposal

		burnDeposits bool
		tally        govv1.TallyResult

		senderPk cryptotypes.PrivKey
	)

	BeforeEach(func() {
		s = new(BurnCoinsTestSuite)
		s.SetupTest()

		senderPk = s.keyring.GetPrivKey(0)

		// fund voters
		Expect(s.network.FundAccountWithBaseDenom(s.keyringVals.GetAccAddr(0), math.NewIntWithDecimal(5, 18))).To(BeNil())
		Expect(s.network.FundAccountWithBaseDenom(s.keyringVals.GetAccAddr(1), math.NewIntWithDecimal(5, 18))).To(BeNil())
		Expect(s.network.FundAccountWithBaseDenom(s.keyringVals.GetAccAddr(2), math.NewIntWithDecimal(5, 18))).To(BeNil())

		// get current Governance Params
		resGov, err := s.network.GetGovClient().Params(s.network.GetContext(), &govv1.QueryParamsRequest{})
		Expect(err).To(BeNil())
		govParams = resGov.Params

		// get initial Community Pool balance
		resDistr, err := s.network.GetDistrClient().CommunityPool(s.network.GetContext(), &distrtypes.QueryCommunityPoolRequest{})
		Expect(err).To(BeNil())
		cpInitialBal = resDistr.Pool

		// prepare proposal
		govDefaultParams := govv1.DefaultParams()
		proposalMsg := &govv1.MsgUpdateParams{Authority: authority, Params: govDefaultParams}
		proposalID, err := utils.SubmitProposal(s.factory, s.network, senderPk, "Reset gov params - no quorum", proposalMsg)
		Expect(err).To(BeNil())

		proposal, err = s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalID)
		Expect(err).To(BeNil())
		err = utils.CheckProposalStatus(s.network, proposalID, govv1.StatusVotingPeriod)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		resDistr, err := s.network.GetDistrClient().CommunityPool(s.network.GetContext(), &distrtypes.QueryCommunityPoolRequest{})
		Expect(err).To(BeNil())

		println("Community pool balances:")
		println(fmt.Sprintf(" before > %s", cpInitialBal.String()))
		println(fmt.Sprintf("  after > %s", resDistr.Pool.String()))

		if burnDeposits {
			Expect(resDistr.Pool.Empty()).To(BeFalse())
			Expect(resDistr.Pool.IsZero()).To(BeFalse())
			Expect(resDistr.Pool.IsZero()).To(BeFalse())
			Expect(sdk.NormalizeCoins(resDistr.Pool).IsAllGTE(sdk.NormalizeCoins(cpInitialBal).Add(proposal.TotalDeposit...))).To(BeTrue())
			println("should be increased by: " + proposal.TotalDeposit[0].String())
		} else {
			println("should not changed")
			Expect(sdk.NormalizeCoins(resDistr.Pool).IsAllLT(sdk.NormalizeCoins(cpInitialBal).Add(proposal.TotalDeposit...))).To(BeTrue())
		}
	})

	It("No quorum", func() {
		burnDeposits = govParams.BurnVoteQuorum
		// vote for proposal, but use 1/4 of power to fail on quorum
		msgVote := govv1.NewMsgVote(s.keyringVals.GetAccAddr(0), proposal.Id, govv1.OptionAbstain, "")
		voteRes, err := s.factory.CommitCosmosTx(s.keyringVals.GetPrivKey(0), commonfactory.CosmosTxArgs{Msgs: []sdk.Msg{msgVote}})
		Expect(err).To(BeNil())
		Expect(voteRes.IsOK()).To(BeTrue())

		err = utils.WaitVotingPeriod(s.network)
		Expect(err).To(BeNil())

		err = utils.CheckProposalStatus(s.network, proposal.Id, govv1.StatusRejected)
		Expect(err).To(BeNil())

		propRes, err := s.network.GetGovClient().Proposal(s.network.GetContext(), &govv1.QueryProposalRequest{ProposalId: proposal.Id})
		Expect(err).To(BeNil())
		tally = *propRes.Proposal.FinalTallyResult
		Expect(tally.YesCount).NotTo(BeEmpty())
		Expect(tally.YesCount).To(Equal("0"))
		Expect(tally.NoWithVetoCount).To(Equal("0"))
		Expect(tally.NoCount).To(Equal("0"))
		Expect(tally.AbstainCount).To(Equal("1000000000000000000"))

		println(fmt.Sprintf("Proposal #%d:", proposal.Id))
		println(fmt.Sprintf(" > yes:     %s", tally.YesCount))
		println(fmt.Sprintf(" > veto:    %s", tally.NoWithVetoCount))
		println(fmt.Sprintf(" > no:      %s", tally.NoCount))
		println(fmt.Sprintf(" > abstain: %s", tally.AbstainCount))
	})

	It("Quorum with NoWithVeto", func() {
		burnDeposits = govParams.BurnVoteVeto
		// vote for proposal, use 3/4 of power to meet quorum requirements
		msgVote1 := govv1.NewMsgVote(s.keyringVals.GetAccAddr(0), proposal.Id, govv1.OptionNoWithVeto, "")
		voteRes1, err := s.factory.CommitCosmosTx(s.keyringVals.GetPrivKey(0), commonfactory.CosmosTxArgs{Msgs: []sdk.Msg{msgVote1}})
		Expect(err).To(BeNil())
		Expect(voteRes1.IsOK()).To(BeTrue())

		msgVote2 := govv1.NewMsgVote(s.keyringVals.GetAccAddr(1), proposal.Id, govv1.OptionNoWithVeto, "")
		voteRes2, err := s.factory.CommitCosmosTx(s.keyringVals.GetPrivKey(1), commonfactory.CosmosTxArgs{Msgs: []sdk.Msg{msgVote2}})
		Expect(err).To(BeNil())
		Expect(voteRes2.IsOK()).To(BeTrue())

		msgVote3 := govv1.NewMsgVote(s.keyringVals.GetAccAddr(2), proposal.Id, govv1.OptionNoWithVeto, "")
		voteRes3, err := s.factory.CommitCosmosTx(s.keyringVals.GetPrivKey(2), commonfactory.CosmosTxArgs{Msgs: []sdk.Msg{msgVote3}})
		Expect(err).To(BeNil())
		Expect(voteRes3.IsOK()).To(BeTrue())

		err = utils.WaitVotingPeriod(s.network)
		Expect(err).To(BeNil())

		err = utils.CheckProposalStatus(s.network, proposal.Id, govv1.StatusRejected)
		Expect(err).To(BeNil())

		propRes, err := s.network.GetGovClient().Proposal(s.network.GetContext(), &govv1.QueryProposalRequest{ProposalId: proposal.Id})
		Expect(err).To(BeNil())
		tally = *propRes.Proposal.FinalTallyResult
		Expect(tally.YesCount).NotTo(BeEmpty())
		Expect(tally.YesCount).To(Equal("0"))
		Expect(tally.NoWithVetoCount).To(Equal("3000000000000000000"))
		Expect(tally.NoCount).To(Equal("0"))
		Expect(tally.AbstainCount).To(Equal("0"))

		println(fmt.Sprintf("Proposal #%d:", proposal.Id))
		println(fmt.Sprintf(" > yes:     %s", tally.YesCount))
		println(fmt.Sprintf(" > veto:    %s", tally.NoWithVetoCount))
		println(fmt.Sprintf(" > no:      %s", tally.NoCount))
		println(fmt.Sprintf(" > abstain: %s", tally.AbstainCount))
	})

	It("Quorum Yes", func() {
		burnDeposits = false
		// vote for proposal, use 3/4 of power to meet quorum requirements
		msgVote1 := govv1.NewMsgVote(s.keyringVals.GetAccAddr(0), proposal.Id, govv1.OptionYes, "")
		voteRes1, err := s.factory.CommitCosmosTx(s.keyringVals.GetPrivKey(0), commonfactory.CosmosTxArgs{Msgs: []sdk.Msg{msgVote1}})
		Expect(err).To(BeNil())
		Expect(voteRes1.IsOK()).To(BeTrue())

		msgVote2 := govv1.NewMsgVote(s.keyringVals.GetAccAddr(1), proposal.Id, govv1.OptionYes, "")
		voteRes2, err := s.factory.CommitCosmosTx(s.keyringVals.GetPrivKey(1), commonfactory.CosmosTxArgs{Msgs: []sdk.Msg{msgVote2}})
		Expect(err).To(BeNil())
		Expect(voteRes2.IsOK()).To(BeTrue())

		msgVote3 := govv1.NewMsgVote(s.keyringVals.GetAccAddr(2), proposal.Id, govv1.OptionYes, "")
		voteRes3, err := s.factory.CommitCosmosTx(s.keyringVals.GetPrivKey(2), commonfactory.CosmosTxArgs{Msgs: []sdk.Msg{msgVote3}})
		Expect(err).To(BeNil())
		Expect(voteRes3.IsOK()).To(BeTrue())

		err = utils.WaitVotingPeriod(s.network)
		Expect(err).To(BeNil())

		err = utils.CheckProposalStatus(s.network, proposal.Id, govv1.StatusPassed)
		Expect(err).To(BeNil())

		propRes, err := s.network.GetGovClient().Proposal(s.network.GetContext(), &govv1.QueryProposalRequest{ProposalId: proposal.Id})
		Expect(err).To(BeNil())
		tally = *propRes.Proposal.FinalTallyResult
		Expect(burnDeposits).To(Equal(govParams.BurnVoteQuorum))
		Expect(tally.YesCount).NotTo(BeEmpty())
		Expect(tally.YesCount).To(Equal("3000000000000000000"))
		Expect(tally.NoWithVetoCount).To(Equal("0"))
		Expect(tally.NoCount).To(Equal("0"))
		Expect(tally.AbstainCount).To(Equal("0"))

		println(fmt.Sprintf("Proposal #%d:", proposal.Id))
		println(fmt.Sprintf(" > yes:     %s", tally.YesCount))
		println(fmt.Sprintf(" > veto:    %s", tally.NoWithVetoCount))
		println(fmt.Sprintf(" > no:      %s", tally.NoCount))
		println(fmt.Sprintf(" > abstain: %s", tally.AbstainCount))
	})

	// TODO
	// It("Low deposit", func() {
	// })
})
