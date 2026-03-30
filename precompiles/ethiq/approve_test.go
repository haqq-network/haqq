package ethiq_test

import (
	"fmt"
	"math/big"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

// unpackBool unpacks a single bool return value from ABI-encoded bytes.
func unpackBool(s *PrecompileTestSuite, methodName string, bz []byte) bool {
	results, err := s.precompile.ABI.Methods[methodName].Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Len(results, 1)
	result, ok := results[0].(bool)
	s.Require().True(ok)
	return result
}

// ---------------------------------------------------------------------------
// Approve
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestApprove() {
	var ctx sdk.Context
	method := s.precompile.Methods["approve"]

	testCases := []struct {
		name        string
		malleate    func() []any
		postCheck   func(bz []byte)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []any { return []any{} },
			func(_ []byte) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 3, 0),
		},
		{
			"fail - invalid message type URL",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					abi.MaxUint256,
					[]string{"invalid/url"},
				}
			},
			func(_ []byte) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidMsgType, "ethiq", "invalid/url"),
		},
		{
			"success - unlimited allowance (MaxUint256 → nil coin)",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					abi.MaxUint256, // coin == nil → no limit
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(bz []byte) {
				s.Require().True(unpackBool(s, "approve", bz))
			},
			200000, false, "",
		},
		{
			"success - positive coin amount",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(bz []byte) {
				s.Require().True(unpackBool(s, "approve", bz))
			},
			200000, false, "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			bz, err := s.precompile.Approve(ctx, s.keyring.GetAddr(0), s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				tc.postCheck(bz)
			}
			_ = contract
		})
	}
}

// TestApproveDeletesGrant verifies that passing a zero-amount coin deletes an existing grant.
func (s *PrecompileTestSuite) TestApproveDeletesGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	granter := s.keyring.GetAddr(0)
	grantee := s.keyring.GetAddr(1)
	method := s.precompile.Methods["approve"]

	// First, create the grant
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, granter, s.precompile, 200000)
	_, createErr := s.precompile.Approve(ctx, granter, s.network.GetStateDB(), &method, []any{
		grantee,
		big.NewInt(1e18),
		[]string{ethiq.MintHaqqMsgURL},
	})
	s.Require().NoError(createErr)

	// Verify the grant exists
	savedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), ethiq.MintHaqqMsgURL)
	s.Require().NotNil(savedAuthz)

	// Now send zero amount → should delete the grant
	_, deleteErr := s.precompile.Approve(ctx, granter, s.network.GetStateDB(), &method, []any{
		grantee,
		big.NewInt(0),
		[]string{ethiq.MintHaqqMsgURL},
	})
	s.Require().NoError(deleteErr)

	// Verify the grant is gone (GetAuthorization returns nil when not found)
	deletedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), ethiq.MintHaqqMsgURL)
	s.Require().Nil(deletedAuthz)
	_ = contract
}

// ---------------------------------------------------------------------------
// ApproveApplicationID
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestApproveApplicationID() {
	var ctx sdk.Context
	method := s.precompile.Methods[ethiq.ApproveApplicationIDMethod]

	testCases := []struct {
		name        string
		malleate    func() []any
		postCheck   func(bz []byte)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []any { return []any{} },
			func(_ []byte) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 3, 0),
		},
		{
			"fail - invalid method URL",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(0),
					[]string{"invalid/url"},
				}
			},
			func(_ []byte) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidMsgType, "ethiq", "invalid/url"),
		},
		{
			"success - valid application ID and method",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(0),
					[]string{ethiq.MsgMintHaqqByApplicationMsgURL},
				}
			},
			func(bz []byte) {
				s.Require().True(unpackBool(s, ethiq.ApproveApplicationIDMethod, bz))
			},
			200000, false, "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			bz, err := s.precompile.ApproveApplicationID(ctx, s.keyring.GetAddr(0), s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				tc.postCheck(bz)
			}
			_ = contract
		})
	}
}

// ---------------------------------------------------------------------------
// Revoke
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestRevoke() {
	var ctx sdk.Context
	method := s.precompile.Methods["revoke"]

	testCases := []struct {
		name        string
		malleate    func() []any
		setup       func(granter, grantee sdk.AccAddress)
		postCheck   func(bz []byte)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []any { return []any{} },
			func(_, _ sdk.AccAddress) {},
			func(_ []byte) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 2, 0),
		},
		{
			"fail - invalid message type",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					[]string{"invalid/url"},
				}
			},
			func(_, _ sdk.AccAddress) {},
			func(_ []byte) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidMsgType, "ethiq", "invalid/url"),
		},
		{
			"success - revoke existing grant",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(granter, grantee sdk.AccAddress) {
				authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: nil}
				expiration := time.Now().Add(time.Hour).UTC()
				err := s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee, granter, authzGrant, &expiration)
				s.Require().NoError(err)
			},
			func(bz []byte) {
				s.Require().True(unpackBool(s, "revoke", bz))
			},
			200000, false, "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			granter := sdk.AccAddress(s.keyring.GetAddr(0).Bytes())
			grantee := sdk.AccAddress(s.keyring.GetAddr(1).Bytes())
			tc.setup(granter, grantee)

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			bz, err := s.precompile.Revoke(ctx, s.keyring.GetAddr(0), s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				tc.postCheck(bz)
			}
			_ = contract
		})
	}
}

// ---------------------------------------------------------------------------
// RevokeApplicationID
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestRevokeApplicationID() {
	var ctx sdk.Context
	method := s.precompile.Methods[ethiq.RevokeApplicationIDMethod]

	testCases := []struct {
		name        string
		malleate    func() []any
		setup       func(granter, grantee sdk.AccAddress)
		postCheck   func(bz []byte)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []any { return []any{} },
			func(_, _ sdk.AccAddress) {},
			func(_ []byte) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 2, 0),
		},
		{
			"fail - invalid method URL",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					[]string{"invalid/url"},
				}
			},
			func(_, _ sdk.AccAddress) {},
			func(_ []byte) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidMsgType, "ethiq", "invalid/url"),
		},
		{
			"success - revoke existing application ID grant",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					[]string{ethiq.MsgMintHaqqByApplicationMsgURL},
				}
			},
			func(granter, grantee sdk.AccAddress) {
				authzGrant := &ethiqtypes.MintHaqqByApplicationIDAuthorization{
					ApplicationsList: []uint64{0},
				}
				expiration := time.Now().Add(time.Hour).UTC()
				err := s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee, granter, authzGrant, &expiration)
				s.Require().NoError(err)
			},
			func(bz []byte) {
				s.Require().True(unpackBool(s, ethiq.RevokeApplicationIDMethod, bz))
			},
			200000, false, "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			granter := sdk.AccAddress(s.keyring.GetAddr(0).Bytes())
			grantee := sdk.AccAddress(s.keyring.GetAddr(1).Bytes())
			tc.setup(granter, grantee)

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			bz, err := s.precompile.RevokeApplicationID(ctx, s.keyring.GetAddr(0), s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				tc.postCheck(bz)
			}
			_ = contract
		})
	}
}

// ---------------------------------------------------------------------------
// IncreaseAllowance
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestIncreaseAllowance() {
	var ctx sdk.Context
	method := s.precompile.Methods["increaseAllowance"]

	testCases := []struct {
		name        string
		malleate    func() []any
		setup       func(granter, grantee sdk.AccAddress)
		postCheck   func(ctx sdk.Context, granter, grantee sdk.AccAddress)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []any { return []any{} },
			func(_, _ sdk.AccAddress) {},
			func(_ sdk.Context, _, _ sdk.AccAddress) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 3, 0),
		},
		{
			"fail - invalid method URL",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
					[]string{"invalid/url"},
				}
			},
			func(_, _ sdk.AccAddress) {},
			func(_ sdk.Context, _, _ sdk.AccAddress) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidMsgType, "ethiq", "invalid/url"),
		},
		{
			"fail - no existing grant",
			func() []any {
				return []any{
					utiltx.GenerateAddress(),
					big.NewInt(1e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(_, _ sdk.AccAddress) {},
			func(_ sdk.Context, _, _ sdk.AccAddress) {},
			200000, true, "",
		},
		{
			"success - increase allowance on existing limited grant",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(granter, grantee sdk.AccAddress) {
				coin := sdk.NewCoin(utils.BaseDenom, math.NewInt(2e18))
				authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: &coin}
				expiration := time.Now().Add(time.Hour).UTC()
				err := s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee, granter, authzGrant, &expiration)
				s.Require().NoError(err)
			},
			func(ctx sdk.Context, granter, grantee sdk.AccAddress) {
				savedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee, granter, ethiq.MintHaqqMsgURL)
				s.Require().NotNil(savedAuthz)
				mintAuthz, ok := savedAuthz.(*ethiqtypes.MintHaqqAuthorization)
				s.Require().True(ok)
				s.Require().Equal(math.NewInt(3e18), mintAuthz.SpendLimit.Amount)
			},
			200000, false, "",
		},
		{
			"success - no-op when existing grant has no limit",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(granter, grantee sdk.AccAddress) {
				authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: nil}
				expiration := time.Now().Add(time.Hour).UTC()
				err := s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee, granter, authzGrant, &expiration)
				s.Require().NoError(err)
			},
			func(ctx sdk.Context, granter, grantee sdk.AccAddress) {
				savedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee, granter, ethiq.MintHaqqMsgURL)
				s.Require().NotNil(savedAuthz)
				mintAuthz, ok := savedAuthz.(*ethiqtypes.MintHaqqAuthorization)
				s.Require().True(ok)
				s.Require().Nil(mintAuthz.SpendLimit)
			},
			200000, false, "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			granter := sdk.AccAddress(s.keyring.GetAddr(0).Bytes())
			grantee := sdk.AccAddress(s.keyring.GetAddr(1).Bytes())
			tc.setup(granter, grantee)

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			_, err := s.precompile.IncreaseAllowance(ctx, s.keyring.GetAddr(0), s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				tc.postCheck(ctx, granter, grantee)
			}
			_ = contract
		})
	}
}

// ---------------------------------------------------------------------------
// DecreaseAllowance
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestDecreaseAllowance() {
	var ctx sdk.Context
	method := s.precompile.Methods["decreaseAllowance"]

	testCases := []struct {
		name        string
		malleate    func() []any
		setup       func(granter, grantee sdk.AccAddress)
		postCheck   func(ctx sdk.Context, granter, grantee sdk.AccAddress)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []any { return []any{} },
			func(_, _ sdk.AccAddress) {},
			func(_ sdk.Context, _, _ sdk.AccAddress) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 3, 0),
		},
		{
			"fail - invalid method URL",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
					[]string{"invalid/url"},
				}
			},
			func(_, _ sdk.AccAddress) {},
			func(_ sdk.Context, _, _ sdk.AccAddress) {},
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidMsgType, "ethiq", "invalid/url"),
		},
		{
			"fail - no existing grant",
			func() []any {
				return []any{
					utiltx.GenerateAddress(),
					big.NewInt(1e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(_, _ sdk.AccAddress) {},
			func(_ sdk.Context, _, _ sdk.AccAddress) {},
			200000, true, "",
		},
		{
			"fail - decrease exceeds current limit",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(3e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(granter, grantee sdk.AccAddress) {
				coin := sdk.NewCoin(utils.BaseDenom, math.NewInt(2e18))
				authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: &coin}
				expiration := time.Now().Add(time.Hour).UTC()
				err := s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee, granter, authzGrant, &expiration)
				s.Require().NoError(err)
			},
			func(_ sdk.Context, _, _ sdk.AccAddress) {},
			200000, true, "decrease amount",
		},
		{
			"success - decrease within limit",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(granter, grantee sdk.AccAddress) {
				coin := sdk.NewCoin(utils.BaseDenom, math.NewInt(3e18))
				authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: &coin}
				expiration := time.Now().Add(time.Hour).UTC()
				err := s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee, granter, authzGrant, &expiration)
				s.Require().NoError(err)
			},
			func(ctx sdk.Context, granter, grantee sdk.AccAddress) {
				savedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee, granter, ethiq.MintHaqqMsgURL)
				s.Require().NotNil(savedAuthz)
				mintAuthz, ok := savedAuthz.(*ethiqtypes.MintHaqqAuthorization)
				s.Require().True(ok)
				s.Require().Equal(math.NewInt(2e18), mintAuthz.SpendLimit.Amount)
			},
			200000, false, "",
		},
		{
			"success - decrease to zero deletes grant",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(2e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(granter, grantee sdk.AccAddress) {
				coin := sdk.NewCoin(utils.BaseDenom, math.NewInt(2e18))
				authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: &coin}
				expiration := time.Now().Add(time.Hour).UTC()
				err := s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee, granter, authzGrant, &expiration)
				s.Require().NoError(err)
			},
			func(ctx sdk.Context, granter, grantee sdk.AccAddress) {
				deletedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee, granter, ethiq.MintHaqqMsgURL)
				s.Require().Nil(deletedAuthz)
			},
			200000, false, "",
		},
		{
			"success - no-op when existing grant has no limit",
			func() []any {
				return []any{
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
			},
			func(granter, grantee sdk.AccAddress) {
				authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: nil}
				expiration := time.Now().Add(time.Hour).UTC()
				err := s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee, granter, authzGrant, &expiration)
				s.Require().NoError(err)
			},
			func(ctx sdk.Context, granter, grantee sdk.AccAddress) {
				savedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee, granter, ethiq.MintHaqqMsgURL)
				s.Require().NotNil(savedAuthz)
				mintAuthz, ok := savedAuthz.(*ethiqtypes.MintHaqqAuthorization)
				s.Require().True(ok)
				s.Require().Nil(mintAuthz.SpendLimit)
			},
			200000, false, "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			granter := sdk.AccAddress(s.keyring.GetAddr(0).Bytes())
			grantee := sdk.AccAddress(s.keyring.GetAddr(1).Bytes())
			tc.setup(granter, grantee)

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)
			_, err := s.precompile.DecreaseAllowance(ctx, s.keyring.GetAddr(0), s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				tc.postCheck(ctx, granter, grantee)
			}
			_ = contract
		})
	}
}
