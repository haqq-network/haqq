package types_test

import (
	"strings"
	"testing"

	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	erc20types "github.com/evmos/evmos/v14/x/erc20/types"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
)

type TokenPairTestSuite struct {
	suite.Suite
}

func TestTokenPairSuite(t *testing.T) {
	suite.Run(t, new(TokenPairTestSuite))
}

func (suite *TokenPairTestSuite) TestTokenPairNew() {
	testCases := []struct {
		msg          string
		erc20Address common.Address
		denom        string
		owner        erc20types.Owner
		expectPass   bool
	}{
		{msg: "Register token pair - invalid starts with number", erc20Address: utiltx.GenerateAddress(), denom: "1test", owner: erc20types.OWNER_MODULE, expectPass: false},
		{msg: "Register token pair - invalid char '('", erc20Address: utiltx.GenerateAddress(), denom: "(test", owner: erc20types.OWNER_MODULE, expectPass: false},
		{msg: "Register token pair - invalid char '^'", erc20Address: utiltx.GenerateAddress(), denom: "^test", owner: erc20types.OWNER_MODULE, expectPass: false},
		// TODO: (guille) should the "\" be allowed to support unicode names?
		{msg: "Register token pair - invalid char '\\'", erc20Address: utiltx.GenerateAddress(), denom: "-test", owner: erc20types.OWNER_MODULE, expectPass: false},
		// Invalid length
		{msg: "Register token pair - invalid length token (0)", erc20Address: utiltx.GenerateAddress(), denom: "", owner: erc20types.OWNER_MODULE, expectPass: false},
		{msg: "Register token pair - invalid length token (1)", erc20Address: utiltx.GenerateAddress(), denom: "a", owner: erc20types.OWNER_MODULE, expectPass: false},
		{msg: "Register token pair - invalid length token (128)", erc20Address: utiltx.GenerateAddress(), denom: strings.Repeat("a", 129), owner: erc20types.OWNER_MODULE, expectPass: false},
		{msg: "Register token pair - pass", erc20Address: utiltx.GenerateAddress(), denom: "test", owner: erc20types.OWNER_MODULE, expectPass: true},
	}

	for i, tc := range testCases {
		tp := erc20types.NewTokenPair(tc.erc20Address, tc.denom, tc.owner)
		err := tp.Validate()

		if tc.expectPass {
			suite.Require().NoError(err, "valid test %d failed: %s, %v", i, tc.msg)
		} else {
			suite.Require().Error(err, "invalid test %d passed: %s, %v", i, tc.msg)
		}
	}
}

func (suite *TokenPairTestSuite) TestTokenPair() {
	testCases := []struct {
		msg        string
		pair       erc20types.TokenPair
		expectPass bool
	}{
		{msg: "Register token pair - invalid address (no hex)", pair: erc20types.TokenPair{"0x5dCA2483280D9727c80b5518faC4556617fb19ZZ", "test", true, erc20types.OWNER_MODULE}, expectPass: false},
		{msg: "Register token pair - invalid address (invalid length 1)", pair: erc20types.TokenPair{"0x5dCA2483280D9727c80b5518faC4556617fb19", "test", true, erc20types.OWNER_MODULE}, expectPass: false},
		{msg: "Register token pair - invalid address (invalid length 2)", pair: erc20types.TokenPair{"0x5dCA2483280D9727c80b5518faC4556617fb194FFF", "test", true, erc20types.OWNER_MODULE}, expectPass: false},
		{msg: "pass", pair: erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", true, erc20types.OWNER_MODULE}, expectPass: true},
	}

	for i, tc := range testCases {
		err := tc.pair.Validate()

		if tc.expectPass {
			suite.Require().NoError(err, "valid test %d failed: %s, %v", i, tc.msg)
		} else {
			suite.Require().Error(err, "invalid test %d passed: %s, %v", i, tc.msg)
		}
	}
}

func (suite *TokenPairTestSuite) TestGetID() {
	addr := utiltx.GenerateAddress()
	denom := "test"
	pair := erc20types.NewTokenPair(addr, denom, erc20types.OWNER_MODULE)
	id := pair.GetID()
	expID := tmhash.Sum([]byte(addr.String() + "|" + denom))
	suite.Require().Equal(expID, id)
}

func (suite *TokenPairTestSuite) TestGetERC20Contract() {
	expAddr := utiltx.GenerateAddress()
	denom := "test"
	pair := erc20types.NewTokenPair(expAddr, denom, erc20types.OWNER_MODULE)
	addr := pair.GetERC20Contract()
	suite.Require().Equal(expAddr, addr)
}

func (suite *TokenPairTestSuite) TestIsNativeCoin() {
	testCases := []struct {
		name       string
		pair       erc20types.TokenPair
		expectPass bool
	}{
		{
			"no owner",
			erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", true, erc20types.OWNER_UNSPECIFIED},
			false,
		},
		{
			"external ERC20 owner",
			erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", true, erc20types.OWNER_EXTERNAL},
			false,
		},
		{
			"pass",
			erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", true, erc20types.OWNER_MODULE},
			true,
		},
	}

	for _, tc := range testCases {
		res := tc.pair.IsNativeCoin()
		if tc.expectPass {
			suite.Require().True(res, tc.name)
		} else {
			suite.Require().False(res, tc.name)
		}
	}
}

func (suite *TokenPairTestSuite) TestIsNativeERC20() {
	testCases := []struct {
		name       string
		pair       erc20types.TokenPair
		expectPass bool
	}{
		{
			"no owner",
			erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", true, erc20types.OWNER_UNSPECIFIED},
			false,
		},
		{
			"module owner",
			erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", true, erc20types.OWNER_MODULE},
			false,
		},
		{
			"pass",
			erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", true, erc20types.OWNER_EXTERNAL},
			true,
		},
	}

	for _, tc := range testCases {
		res := tc.pair.IsNativeERC20()
		if tc.expectPass {
			suite.Require().True(res, tc.name)
		} else {
			suite.Require().False(res, tc.name)
		}
	}
}
