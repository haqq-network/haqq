package gov

import (
	"context"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	haqqnetwork "github.com/haqq-network/haqq/testutil/network"
	"github.com/stretchr/testify/suite"
	"github.com/tharsis/ethermint/testutil/network"
	"testing"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func NewIntegrationTestSuite(cfg network.Config) *IntegrationTestSuite {
	return &IntegrationTestSuite{cfg: cfg}
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	var err error
	baseDir := s.T().TempDir()
	s.network, err = network.New(s.T(), baseDir, s.cfg)
	s.Require().NoError(err)

	_, err = s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestCommunityProposals() {
	//test
	val := s.network.Validators[0]

	testCases := []struct {
		name     string
		valid    bool
		proposal govtypes.Content
	}{
		{
			name:  "software-upgrade",
			valid: true,
			proposal: upgradetypes.NewSoftwareUpgradeProposal("test", "test", upgradetypes.Plan{
				Name:   "test",
				Height: 10,
				Info:   "test",
			}),
		},
		{
			name:  "community-pool-spend",
			valid: false,
			proposal: types.NewCommunityPoolSpendProposal("Test", "description", val.Address, sdk.NewCoins(
				sdk.NewCoin("ISLM", sdk.NewInt(1)),
			)),
		},
	}

	for i, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			submitProposalMsg, err := govtypes.NewMsgSubmitProposal(tc.proposal, sdk.NewCoins(), val.Address)
			s.Require().NoError(err)

			fee := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(100)))
			txBuilder := s.cfg.TxConfig.NewTxBuilder()
			s.Require().NoError(txBuilder.SetMsgs(submitProposalMsg))
			txBuilder.SetFeeAmount(fee)
			txBuilder.SetMemo("test")
			txBuilder.SetGasLimit(2000000)

			kb := val.ClientCtx.Keyring
			s.Require().NoError(err)

			txFactory := tx.Factory{}
			txFactory = txFactory.
				WithChainID(s.cfg.ChainID).
				WithKeybase(kb).
				WithTxConfig(s.cfg.TxConfig).
				WithSequence(uint64(i + 1))

			err = tx.Sign(txFactory, "node0", txBuilder, true)
			s.Require().NoError(err)

			txBytes, err := s.cfg.TxConfig.TxEncoder()(txBuilder.GetTx())
			s.Require().NoError(err)

			result, err := val.RPCClient.BroadcastTxSync(context.Background(), txBytes)
			s.Require().NoError(err)

			err = s.network.WaitForNextBlock()
			s.Require().NoError(err)

			if tc.valid {
				s.Require().Equal("[]", result.Log)
				s.Require().Equal(uint32(0), result.Code)
			} else {
				s.Require().NotEqual(uint32(0), result.Code)
			}
		})
	}
}

func TestDisabledCommunityProposals(t *testing.T) {
	cfg := network.DefaultConfig()
	encCfg := simapp.MakeTestEncodingConfig()
	cfg.AppConstructor = haqqnetwork.NewAppConstructor(encCfg)
	cfg.NumValidators = 1

	suite.Run(t, NewIntegrationTestSuite(cfg))
}
