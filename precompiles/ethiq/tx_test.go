package ethiq_test

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

func (s *PrecompileTestSuite) TestMintHaqq() {
	var ctx sdk.Context
	method := s.precompile.Methods[ethiq.MintHaqq]

	testCases := []struct {
		name        string
		malleate    func() []any
		postCheck   func()
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []any {
				return []any{}
			},
			func() {},
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
			func() {},
			200000,
			true,
			"invalid sender",
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
			func() {},
			200000,
			true,
			"origin address",
		},
		{
			"fail - zero amount",
			func() []any {
				return []any{
					s.keyring.GetAddr(0),
					s.keyring.GetAddr(1),
					big.NewInt(0),
				}
			},
			func() {},
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
			_, err := s.precompile.MintHaqq(ctx, s.keyring.GetAddr(0), contract, s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				tc.postCheck()
			}
		})
	}
}

func (s *PrecompileTestSuite) TestMintHaqqWithFunds() {
	s.SetupTest()
	ctx := s.network.GetContext()

	sender := s.keyring.GetAddr(0)
	receiver := s.keyring.GetAddr(1)
	burnAmount := big.NewInt(1e18)

	// Fund sender with aISLM
	senderAccAddr := sdk.AccAddress(sender.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin("aISLM", sdkmath.NewIntFromBigInt(burnAmount).MulRaw(2)))
	err := s.network.App.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins)
	s.Require().NoError(err)
	err = s.network.App.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, senderAccAddr, coins)
	s.Require().NoError(err)

	method := s.precompile.Methods[ethiq.MintHaqq]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, sender, s.precompile, 200000)

	args := []any{
		sender,
		receiver,
		burnAmount,
	}

	bz, err := s.precompile.MintHaqq(ctx, sender, contract, s.network.GetStateDB(), &method, args)
	s.Require().NoError(err)
	s.Require().NotNil(bz)
}

func (s *PrecompileTestSuite) TestMintHaqqByApplication() {
	var ctx sdk.Context
	method := s.precompile.Methods[ethiq.MintHaqqByApplication]

	testCases := []struct {
		name        string
		malleate    func() []any
		postCheck   func()
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []any {
				return []any{}
			},
			func() {},
			200000,
			true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 2, 0),
		},
		{
			"fail - invalid sender (non-address type)",
			func() []any {
				return []any{
					"not-an-address",
					big.NewInt(0),
				}
			},
			func() {},
			200000,
			true,
			"invalid sender",
		},
		{
			"fail - different origin from sender",
			func() []any {
				differentAddr := utiltx.GenerateAddress()
				return []any{
					differentAddr,
					big.NewInt(0),
				}
			},
			func() {},
			200000,
			true,
			"origin address",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			_, err := s.precompile.MintHaqqByApplication(ctx, s.keyring.GetAddr(0), contract, s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				tc.postCheck()
			}
		})
	}
}
