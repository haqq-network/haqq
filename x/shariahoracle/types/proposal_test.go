package types_test

import (
	"strings"
	"testing"

	length "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
	"github.com/stretchr/testify/suite"
)

type ProposalTestSuite struct {
	suite.Suite
}

func TestProposalTestSuite(t *testing.T) {
	suite.Run(t, new(ProposalTestSuite))
}

func (suite *ProposalTestSuite) TestKeysTypes() {
	suite.Require().Equal("shariahoracle", (&types.MintCACProposal{}).ProposalRoute())
	suite.Require().Equal("MintCAC", (&types.MintCACProposal{}).ProposalType())
	suite.Require().Equal("shariahoracle", (&types.BurnCACProposal{}).ProposalRoute())
	suite.Require().Equal("BurnCAC", (&types.BurnCACProposal{}).ProposalType())
	suite.Require().Equal("shariahoracle", (&types.UpdateCACContractProposal{}).ProposalRoute())
	suite.Require().Equal("UpdateCACContract", (&types.UpdateCACContractProposal{}).ProposalType())
}

func (suite *ProposalTestSuite) TestMintCACProposal() {
	testCases := []struct {
		msg         string
		title       string
		description string
		grantees    []string
		expectPass  bool
	}{
		// Valid tests
		{msg: "Mint CAC - valid grantee address", title: "test", description: "test desc", grantees: []string{utiltx.GenerateAddress().String()}, expectPass: true},
		// Missing params valid
		{msg: "Mint CAC - invalid missing title ", title: "", description: "test desc", grantees: []string{utiltx.GenerateAddress().String()}, expectPass: false},
		{msg: "Mint CAC - invalid missing description ", title: "test", description: "", grantees: []string{utiltx.GenerateAddress().String()}, expectPass: false},
		// Invalid address
		{msg: "Mint CAC - invalid address (no hex)", title: "test", description: "test desc", grantees: []string{""}, expectPass: false},
		{msg: "Mint CAC - invalid address (invalid length 1)", title: "test", description: "test desc", grantees: []string{"0x5dCA2483280D9727c80b5518faC4556617fb194FFF"}, expectPass: false},
		{msg: "Mint CAC - invalid address (invalid prefix)", title: "test", description: "test desc", grantees: []string{"1x5dCA2483280D9727c80b5518faC4556617fb19F"}, expectPass: false},
	}

	for i, tc := range testCases {
		tx := types.NewMintCACProposal(tc.title, tc.description, tc.grantees...)
		err := tx.ValidateBasic()

		if tc.expectPass {
			suite.Require().NoError(err, "valid test %d failed: %s, %v", i, tc.msg)
		} else {
			suite.Require().Error(err, "invalid test %d passed: %s, %v", i, tc.msg)
		}
	}
}

func (suite *ProposalTestSuite) TestBurnCACProposal() {
	testCases := []struct {
		msg         string
		title       string
		description string
		grantees    []string
		expectPass  bool
	}{
		// Valid tests
		{msg: "Burn CAC - valid grantee address", title: "test", description: "test desc", grantees: []string{utiltx.GenerateAddress().String()}, expectPass: true},
		// Missing params valid
		{msg: "Burn CAC - invalid missing title ", title: "", description: "test desc", grantees: []string{utiltx.GenerateAddress().String()}, expectPass: false},
		{msg: "Burn CAC - invalid missing description ", title: "test", description: "", grantees: []string{utiltx.GenerateAddress().String()}, expectPass: false},
		// Invalid address
		{msg: "Burn CAC - invalid address (no hex)", title: "test", description: "test desc", grantees: []string{""}, expectPass: false},
		{msg: "Burn CAC - invalid address (invalid length 1)", title: "test", description: "test desc", grantees: []string{"0x5dCA2483280D9727c80b5518faC4556617fb194FFF"}, expectPass: false},
		{msg: "Burn CAC - invalid address (invalid prefix)", title: "test", description: "test desc", grantees: []string{"1x5dCA2483280D9727c80b5518faC4556617fb19F"}, expectPass: false},
	}

	for i, tc := range testCases {
		tx := types.NewBurnCACProposal(tc.title, tc.description, tc.grantees...)
		err := tx.ValidateBasic()

		if tc.expectPass {
			suite.Require().NoError(err, "valid test %d failed: %s, %v", i, tc.msg)
		} else {
			suite.Require().Error(err, "invalid test %d passed: %s, %v", i, tc.msg)
		}
	}
}

func (suite *ProposalTestSuite) TestUpdateCACContractProposal() {
	testCases := []struct {
		msg         string
		title       string
		description string
		address     string
		expectPass  bool
	}{
		{msg: "Update CAC implementation contract - valid address", title: "test", description: "test desc", address: "0x5dCA2483280D9727c80b5518faC4556617fb194F", expectPass: true}, //gitleaks:allow
		{msg: "Update CAC implementation contract - invalid address", title: "test", description: "test desc", address: "0x123", expectPass: false},

		// Invalid missing params
		{msg: "Update CAC implementation contract - valid missing title", title: "", description: "test desc", address: "test", expectPass: false},
		{msg: "Update CAC implementation contract - valid missing description", title: "test", description: "", address: "test", expectPass: false},
		{msg: "Update CAC implementation contract - invalid missing address", title: "test", description: "test desc", address: "", expectPass: false},

		// Invalid length
		{msg: "Update CAC implementation contract - invalid length (1)", title: "test", description: "test desc", address: "a", expectPass: false},
		{msg: "Update CAC implementation contract - invalid length (128)", title: "test", description: "test desc", address: strings.Repeat("a", 129), expectPass: false},

		{msg: "Update CAC implementation contract - invalid length title (140)", title: strings.Repeat("a", length.MaxTitleLength+1), description: "test desc", address: "test", expectPass: false},
		{msg: "Update CAC implementation contract - invalid length description (5000)", title: "title", description: strings.Repeat("a", length.MaxDescriptionLength+1), address: "test", expectPass: false},
	}

	for i, tc := range testCases {
		tx := types.NewUpdateCACContractProposal(tc.title, tc.description, tc.address)
		err := tx.ValidateBasic()

		if tc.expectPass {
			suite.Require().NoError(err, "valid test %d failed: %s, %v", i, tc.msg)
		} else {
			suite.Require().Error(err, "invalid test %d passed: %s, %v", i, tc.msg)
		}
	}
}
