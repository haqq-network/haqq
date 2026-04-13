package ucdao_test

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
)

func (s *PrecompileTestSuite) TestConvertToHaqq() {
	var ctx sdk.Context
	method := s.precompile.Methods[ucdao.ConvertToHaqqMethod]

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
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 3, 0),
		},
		{
			"fail - invalid sender (non-address type)",
			func() []any {
				return []any{
					"not-an-address",
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
				}
			},
			200000,
			true,
			"invalid sender address",
		},
		{
			"fail - different origin from sender",
			func() []any {
				differentAddr := utiltx.GenerateAddress()
				return []any{
					differentAddr,
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
				}
			},
			200000,
			true,
			"must match transaction origin",
		},
		{
			"fail - nil amount",
			func() []any {
				return []any{
					s.keyring.GetAddr(0),
					s.keyring.GetAddr(1),
					nil,
				}
			},
			200000,
			true,
			"invalid amount",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			_, err := s.precompile.ConvertToHaqq(ctx, s.keyring.GetAddr(0), contract, s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *PrecompileTestSuite) TestTransferOwnership() {
	var ctx sdk.Context
	method := s.precompile.Methods[ucdao.TransferOwnershipMethod]

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
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 2, 0),
		},
		{
			"fail - invalid owner (non-address type)",
			func() []any {
				return []any{
					"not-an-address",
					s.keyring.GetAddr(1),
				}
			},
			200000,
			true,
			"invalid owner address",
		},
		{
			"fail - different origin from owner",
			func() []any {
				differentAddr := utiltx.GenerateAddress()
				return []any{
					differentAddr,
					s.keyring.GetAddr(1),
				}
			},
			200000,
			true,
			"must match transaction origin",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			_, err := s.precompile.TransferOwnership(ctx, s.keyring.GetAddr(0), contract, s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *PrecompileTestSuite) TestTransferOwnershipWithAmount() {
	var ctx sdk.Context
	method := s.precompile.Methods[ucdao.TransferOwnershipWithAmountMethod]

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
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 4, 0),
		},
		{
			"fail - invalid owner (non-address type)",
			func() []any {
				return []any{
					"not-an-address",
					s.keyring.GetAddr(1),
					[]string{"aISLM"},
					[]*big.Int{big.NewInt(1e18)},
				}
			},
			200000,
			true,
			"invalid owner address",
		},
		{
			"fail - different origin from owner",
			func() []any {
				differentAddr := utiltx.GenerateAddress()
				return []any{
					differentAddr,
					s.keyring.GetAddr(1),
					[]string{"aISLM"},
					[]*big.Int{big.NewInt(1e18)},
				}
			},
			200000,
			true,
			"must match transaction origin",
		},
		{
			"fail - denoms and amounts length mismatch",
			func() []any {
				return []any{
					s.keyring.GetAddr(0),
					s.keyring.GetAddr(1),
					[]string{"aISLM", "aHAQQ"},
					[]*big.Int{big.NewInt(1e18)},
				}
			},
			200000,
			true,
			"denoms and amounts length mismatch",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			_, err := s.precompile.TransferOwnershipWithAmount(ctx, s.keyring.GetAddr(0), contract, s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
