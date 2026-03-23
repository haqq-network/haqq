package ethiq_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

// mintHaqqMsgURL is the msg type URL for MintHaqq authorization
var mintHaqqMsgURL = sdk.MsgTypeURL(&ethiqtypes.MsgMintHaqq{})

func (s *PrecompileTestSuite) TestCalculate() {
	var ctx sdk.Context
	method := s.precompile.Methods[ethiq.Calculate]

	testCases := []struct {
		name        string
		malleate    func() []any
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []any {
				return []any{}
			},
			200000,
			true,
			"invalid input arguments",
		},
		{
			"fail - zero amount returns error",
			func() []any {
				return []any{big.NewInt(0)}
			},
			200000,
			true,
			"islm_amount must be positive",
		},
		{
			"success - valid positive amount",
			func() []any {
				return []any{big.NewInt(1e18)}
			},
			200000,
			false,
			"",
		},
		{
			"success - large amount",
			func() []any {
				// 3 million ISLM in aISLM
				amt := new(big.Int)
				amt.SetString("3000000000000000000000000", 10)
				return []any{amt}
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			bz, err := s.precompile.Calculate(ctx, contract, &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(bz)
			}
		})
	}
}

func (s *PrecompileTestSuite) TestCalculateForApplication() {
	var ctx sdk.Context
	method := s.precompile.Methods[ethiq.CalculateForApplication]

	// Find a valid application ID that has non-zero burn amount
	validAppID := uint64(0)
	for i := validAppID; i < ethiqtypes.TotalNumberOfApplications(); i++ {
		app, err := ethiqtypes.GetApplicationByID(i)
		if err == nil && app.BurnAmount.Amount.IsPositive() {
			validAppID = i
			break
		}
	}

	testCases := []struct {
		name        string
		malleate    func() []any
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []any {
				return []any{}
			},
			200000,
			true,
			"invalid input arguments",
		},
		{
			"success - valid application ID",
			func() []any {
				return []any{new(big.Int).SetUint64(validAppID)}
			},
			200000,
			false,
			"",
		},
		{
			"fail - invalid application ID out of range",
			func() []any {
				return []any{big.NewInt(99999)}
			},
			200000,
			true,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			bz, err := s.precompile.CalculateForApplication(ctx, contract, &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(bz)
			}
		})
	}
}

func (s *PrecompileTestSuite) TestAllowance() {
	var ctx sdk.Context
	method := s.precompile.Methods[authorization.AllowanceMethod]

	testCases := []struct {
		name        string
		malleate    func() []any
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []any {
				return []any{}
			},
			200000,
			true,
			"",
		},
		{
			"success - returns 0 when no grant exists",
			func() []any {
				grantee := utiltx.GenerateAddress()
				granter := s.keyring.GetAddr(0)
				return []any{
					grantee,
					granter,
					mintHaqqMsgURL,
				}
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			bz, err := s.precompile.Allowance(ctx, &method, contract, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(bz)
			}
		})
	}
}
