package keeper

import (
	"cosmossdk.io/math"
	"testing"
	"time"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"

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
)

type BurnCoinsTestSuite struct {
	suite.Suite

	network *network.UnitTestNetwork
	handler grpc.Handler
	keyring keyring.Keyring
	factory factory.TxFactory
}

func TestBurnCoinsTestSuite(t *testing.T) {
	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "HAQQ Bank Burn Coins Tests")
}

func (suite *BurnCoinsTestSuite) SetupTest() {
	keys := keyring.New(2)

	twoSeconds, err := time.ParseDuration("2s")
	Expect(err).To(BeNil())
	customGenesis := network.CustomGenesisState{}
	govGenesis := govv1.DefaultGenesisState()
	govGenesis.Params.VotingPeriod = &twoSeconds
	govGenesis.Params.MaxDepositPeriod = &twoSeconds
	customGenesis[govtypes.ModuleName] = govGenesis

	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
		network.WithAmountOfValidators(4),
		network.WithCustomGenesis(customGenesis),
	)
	gh := grpc.NewIntegrationHandler(nw)
	tf := factory.New(nw, gh)

	suite.network = nw
	suite.factory = tf
	suite.handler = gh
	suite.keyring = keys
}

var _ = Describe("Performing Burn tests", Ordered, func() {
	var (
		s            *BurnCoinsTestSuite
		authority    = authtypes.NewModuleAddress(govtypes.ModuleName).String()
		govClient    govv1.QueryClient
		govParams    *govv1.Params
		distrClient  distrtypes.QueryClient
		cpInitialBal sdk.DecCoins

		senderPk     cryptotypes.PrivKey
		voterAccAddr sdk.AccAddress
	)

	BeforeEach(func() {
		s = new(BurnCoinsTestSuite)
		s.SetupTest()

		senderPk = s.keyring.GetPrivKey(0)
		voterAccAddr = sdk.AccAddress(senderPk.PubKey().Address())

		// prepare clients
		govClient = s.network.GetGovClient()
		distrClient = s.network.GetDistrClient()

		// get current Governance Params
		resGov, err := govClient.Params(s.network.GetContext(), &govv1.QueryParamsRequest{})
		Expect(err).To(BeNil())
		govParams = resGov.Params

		// get initial Community Pool balance
		resDistr, err := distrClient.CommunityPool(s.network.GetContext(), &distrtypes.QueryCommunityPoolRequest{})
		Expect(err).To(BeNil())
		cpInitialBal = resDistr.Pool
	})

	It("No quorum", func() {
		// prepare Update Params message for Gov module
		govDefaultParams := govv1.DefaultParams()
		proposalMsg := &govv1.MsgUpdateParams{Authority: authority, Params: govDefaultParams}
		proposalID, err := utils.SubmitProposal(s.factory, s.network, senderPk, "Reset gov params - no quorum", proposalMsg)
		Expect(err).To(BeNil())

		// vote for proposal, but use 1/4 of power to fail on quorum
		msgVote := govv1.NewMsgVoteWeighted(
			voterAccAddr,
			proposalID,
			govv1.WeightedVoteOptions{
				govv1.NewWeightedVoteOption(govv1.OptionYes, math.LegacyNewDecWithPrec(25, 2)),
			},
			"",
		)
		_, err = s.factory.CommitCosmosTx(senderPk, commonfactory.CosmosTxArgs{
			Msgs: []sdk.Msg{msgVote},
		})
		Expect(err).To(BeNil())

		err = utils.WaitVotingPeriod(s.network)
		Expect(err).To(BeNil())

		err = utils.CheckProposalStatus(s.network, proposalID, govv1.StatusRejected)
		Expect(err).To(BeNil())

		gk := s.network.App.GovKeeper
		proposal, err := gk.Proposals.Get(s.network.GetContext(), proposalID)
		Expect(err).To(BeNil())
		passes, burnDeposits, tallyRes, err := gk.Tally(s.network.GetContext(), proposal)
		Expect(err).To(BeNil())
		Expect(passes).To(BeFalse())
		Expect(burnDeposits).To(Equal(govParams.BurnVoteQuorum))
		Expect(tallyRes.YesCount).NotTo(BeEmpty())
		Expect(tallyRes.YesCount).NotTo(Equal("0"))
		Expect(tallyRes.NoWithVetoCount).To(Equal("0"))
		Expect(tallyRes.NoCount).To(Equal("0"))
		Expect(tallyRes.AbstainCount).To(Equal("0"))

		resDistr, err := distrClient.CommunityPool(s.network.GetContext(), &distrtypes.QueryCommunityPoolRequest{})
		Expect(err).To(BeNil())
		if govParams.BurnVoteQuorum {
			Expect(resDistr.Pool.Empty()).To(BeFalse())
			Expect(resDistr.Pool.IsZero()).To(BeFalse())
			Expect(resDistr.Pool.IsZero()).To(BeFalse())
			Expect(resDistr.Pool).To(Equal(sdk.NormalizeCoins(cpInitialBal).Add(proposal.TotalDeposit...)))
		} else {
			Expect(resDistr.Pool.Equal(cpInitialBal)).To(BeTrue())
		}
	})
	It("Quorum with NoWithVeto", func() {
		//TODO
	})
	It("Low deposit", func() {
		//TODO
	})
	It("Quorum Yes", func() {
		//TODO
	})
})
