package ethiq_test

import (
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

// eventNames maps emitted logs back to ABI event names, in order.
func (s *PrecompileTestSuite) eventNames(logs []*ethtypes.Log) []string {
	byID := make(map[common.Hash]string, len(s.precompile.ABI.Events))
	for name, event := range s.precompile.ABI.Events {
		byID[event.ID] = name
	}
	names := make([]string, 0, len(logs))
	for _, l := range logs {
		names = append(names, byID[l.Topics[0]])
	}
	return names
}

// twoPow64 is the first value that no longer fits in uint64. Truncating it with
// big.Int.Uint64 yields 0, which is a valid application ID.
func twoPow64() *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), 64)
}

// TestParseApplicationID covers the uint64 range guard shared by every entry point that
// takes an application ID from calldata.
func (s *PrecompileTestSuite) TestParseApplicationID() {
	maxUint64 := new(big.Int).SetUint64(^uint64(0))

	testCases := []struct {
		name        string
		arg         interface{}
		expID       uint64
		expError    bool
		errContains string
	}{
		{"pass - zero", big.NewInt(0), 0, false, ""},
		{"pass - existing application", big.NewInt(7), 7, false, ""},
		{"pass - max uint64", maxUint64, ^uint64(0), false, ""},
		{
			"fail - 2^64 truncates to 0",
			twoPow64(), 0, true,
			"does not fit in uint64",
		},
		{
			"fail - 2^64+5 truncates to 5",
			new(big.Int).Add(twoPow64(), big.NewInt(5)), 0, true,
			"does not fit in uint64",
		},
		{
			"fail - MaxUint256",
			abi.MaxUint256, 0, true,
			"does not fit in uint64",
		},
		{
			"fail - negative",
			big.NewInt(-1), 0, true,
			"does not fit in uint64",
		},
		{"fail - nil big.Int", (*big.Int)(nil), 0, true, "invalid application id"},
		{"fail - wrong type", "1", 0, true, "invalid application id"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			appID, err := ethiq.ParseApplicationID(tc.arg)
			if tc.expError {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.errContains)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(tc.expID, appID)
		})
	}
}

// TestApplicationIDGuardOnEveryEntryPoint pins the guard on all four call sites that turn a
// uint256 argument into an ApplicationId. Each is given 2^64, which truncates to application 0
// (a real waitlist entry), so an unguarded path would act on someone else's application.
func (s *PrecompileTestSuite) TestApplicationIDGuardOnEveryEntryPoint() {
	s.SetupTest()
	ctx := s.network.GetContext()
	granter := s.keyring.GetAddr(0)
	grantee := s.keyring.GetAddr(1)
	methods := []string{ethiq.MsgMintHaqqByApplicationMsgURL}

	s.Run("approveApplicationID", func() {
		method := s.precompile.Methods[ethiq.ApproveApplicationIDMethod]
		_, err := s.precompile.ApproveApplicationID(
			ctx, granter, s.network.GetStateDB(), &method,
			[]interface{}{grantee, twoPow64(), methods},
		)
		s.Require().Error(err)
		s.Require().ErrorContains(err, "does not fit in uint64")
	})

	s.Run("revokeApplicationID", func() {
		expiration := ctx.BlockTime().Add(time.Hour)
		s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
			ctx,
			sdk.AccAddress(grantee.Bytes()),
			sdk.AccAddress(granter.Bytes()),
			&ethiqtypes.MintHaqqByApplicationIDAuthorization{ApplicationsList: []uint64{0}},
			&expiration,
		))

		method := s.precompile.Methods[ethiq.RevokeApplicationIDMethod]
		_, err := s.precompile.RevokeApplicationID(
			ctx, granter, s.network.GetStateDB(), &method,
			[]interface{}{grantee, twoPow64(), methods},
		)
		s.Require().Error(err)
		s.Require().ErrorContains(err, "does not fit in uint64")

		// The grant for application 0 must survive the rejected call.
		grant, _ := s.network.App.AuthzKeeper.GetAuthorization(
			ctx,
			sdk.AccAddress(grantee.Bytes()),
			sdk.AccAddress(granter.Bytes()),
			ethiq.MsgMintHaqqByApplicationMsgURL,
		)
		s.Require().NotNil(grant)
	})

	s.Run("mintHaqqByApplication", func() {
		_, _, err := ethiq.NewMintHaqqByApplicationMsg([]interface{}{granter, twoPow64()})
		s.Require().Error(err)
		s.Require().ErrorContains(err, "does not fit in uint64")
	})

	s.Run("calculateForApplication", func() {
		_, err := ethiq.NewCalculateForApplicationRequest([]interface{}{twoPow64()})
		s.Require().Error(err)
		s.Require().ErrorContains(err, "does not fit in uint64")
	})
}

// TestApplicationIDEvents checks that approveApplicationID and revokeApplicationID emit the
// resulting allow list. The shared Approval / Revocation events carry neither the application
// ID nor the remaining list, so on their own they cannot describe a partial revoke.
func (s *PrecompileTestSuite) TestApplicationIDEvents() {
	s.SetupTest()
	ctx := s.network.GetContext()
	granter := s.keyring.GetAddr(0)
	grantee := s.keyring.GetAddr(1)
	methods := []string{ethiq.MsgMintHaqqByApplicationMsgURL}

	approveMethod := s.precompile.Methods[ethiq.ApproveApplicationIDMethod]
	revokeMethod := s.precompile.Methods[ethiq.RevokeApplicationIDMethod]

	// decodeIDs reads the (applicationId, uint256[]) data section of the last emitted log.
	decodeIDs := func(eventType string, logs []*ethtypes.Log) (uint64, []uint64) {
		s.Require().NotEmpty(logs)
		event := s.precompile.ABI.Events[eventType]
		last := logs[len(logs)-1]
		s.Require().Equal(event.ID, last.Topics[0])

		args := abi.Arguments{event.Inputs[2], event.Inputs[3]}
		unpacked, err := args.Unpack(last.Data)
		s.Require().NoError(err)
		s.Require().Len(unpacked, 2)

		appID, ok := unpacked[0].(*big.Int)
		s.Require().True(ok)
		rawList, ok := unpacked[1].([]*big.Int)
		s.Require().True(ok)

		list := make([]uint64, len(rawList))
		for i, v := range rawList {
			list[i] = v.Uint64()
		}
		return appID.Uint64(), list
	}

	stateDB := s.network.GetStateDB()
	for _, appID := range []int64{0, 1} {
		_, err := s.precompile.ApproveApplicationID(
			ctx, granter, stateDB, &approveMethod,
			[]interface{}{grantee, big.NewInt(appID), methods},
		)
		s.Require().NoError(err)
	}

	approvedID, approvedList := decodeIDs(ethiq.EventTypeApplicationIDApproval, stateDB.Logs())
	s.Require().Equal(uint64(1), approvedID)
	s.Require().Equal([]uint64{0, 1}, approvedList, "approval event must carry the merged allow list")

	// Partial revoke: the grant survives, so the event has to say what is left.
	stateDB = s.network.GetStateDB()
	_, err := s.precompile.RevokeApplicationID(
		ctx, granter, stateDB, &revokeMethod,
		[]interface{}{grantee, big.NewInt(0), methods},
	)
	s.Require().NoError(err)

	revokedID, remaining := decodeIDs(ethiq.EventTypeApplicationIDRevocation, stateDB.Logs())
	s.Require().Equal(uint64(0), revokedID)
	s.Require().Equal([]uint64{1}, remaining, "partial revoke must report the surviving allow list")
	s.Require().Equal([]string{ethiq.EventTypeApplicationIDRevocation}, s.eventNames(stateDB.Logs()),
		"the grant survives, so Revocation must not be emitted")

	// Full revoke: an empty remaining list is how a deleted grant is reported.
	stateDB = s.network.GetStateDB()
	_, err = s.precompile.RevokeApplicationID(
		ctx, granter, stateDB, &revokeMethod,
		[]interface{}{grantee, big.NewInt(1), methods},
	)
	s.Require().NoError(err)

	revokedID, remaining = decodeIDs(ethiq.EventTypeApplicationIDRevocation, stateDB.Logs())
	s.Require().Equal(uint64(1), revokedID)
	s.Require().Empty(remaining, "revoking the last ID must report an empty allow list")
	s.Require().Equal(
		[]string{authorization.EventTypeRevocation, ethiq.EventTypeApplicationIDRevocation},
		s.eventNames(stateDB.Logs()),
		"removing the last ID also deletes the grant, so both events are emitted",
	)

	grant, _ := s.network.App.AuthzKeeper.GetAuthorization(
		ctx,
		sdk.AccAddress(grantee.Bytes()),
		sdk.AccAddress(granter.Bytes()),
		ethiq.MsgMintHaqqByApplicationMsgURL,
	)
	s.Require().Nil(grant)
}

// TestAllowanceBoundaries covers the two ways a uint256 allowance argument used to leave the
// precompile: the MaxUint256 sentinel (nil coin) and a sum that overflows sdkmath.Int.
func (s *PrecompileTestSuite) TestAllowanceBoundaries() {
	granter := s.keyring.GetKey(0)
	grantee := s.keyring.GetAddr(1)
	methods := []string{ethiq.MintHaqqMsgURL}

	approveMethod := s.precompile.Methods[authorization.ApproveMethod]
	increaseMethod := s.precompile.Methods[authorization.IncreaseAllowanceMethod]
	decreaseMethod := s.precompile.Methods[authorization.DecreaseAllowanceMethod]

	s.Run("increaseAllowance rejects the unlimited sentinel", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		_, err := s.precompile.Approve(ctx, granter.Addr, s.network.GetStateDB(), &approveMethod,
			[]interface{}{grantee, big.NewInt(100), methods})
		s.Require().NoError(err)

		_, err = s.precompile.IncreaseAllowance(ctx, granter.Addr, s.network.GetStateDB(), &increaseMethod,
			[]interface{}{grantee, abi.MaxUint256, methods})
		s.Require().Error(err)
		s.Require().ErrorContains(err, "does not support unlimited amount")
	})

	s.Run("decreaseAllowance rejects the unlimited sentinel", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		_, err := s.precompile.Approve(ctx, granter.Addr, s.network.GetStateDB(), &approveMethod,
			[]interface{}{grantee, big.NewInt(100), methods})
		s.Require().NoError(err)

		_, err = s.precompile.DecreaseAllowance(ctx, granter.Addr, s.network.GetStateDB(), &decreaseMethod,
			[]interface{}{grantee, abi.MaxUint256, methods})
		s.Require().Error(err)
		s.Require().ErrorContains(err, "does not support unlimited amount")
	})

	s.Run("increaseAllowance returns an error instead of panicking on overflow", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		// MaxUint256-1 is the largest amount CheckApprovalArgs turns into a coin; doubling it
		// needs 257 bits, which is where sdkmath.Int used to panic.
		huge := new(big.Int).Sub(abi.MaxUint256, big.NewInt(1))

		_, err := s.precompile.Approve(ctx, granter.Addr, s.network.GetStateDB(), &approveMethod,
			[]interface{}{grantee, huge, methods})
		s.Require().NoError(err)

		s.Require().NotPanics(func() {
			_, err = s.precompile.IncreaseAllowance(ctx, granter.Addr, s.network.GetStateDB(), &increaseMethod,
				[]interface{}{grantee, huge, methods})
		})
		s.Require().Error(err)
		s.Require().ErrorContains(err, "does not fit in 256 bits")

		// The rejected increase must leave the original limit untouched.
		grant, _ := s.network.App.AuthzKeeper.GetAuthorization(
			ctx,
			sdk.AccAddress(grantee.Bytes()),
			granter.AccAddr,
			ethiq.MintHaqqMsgURL,
		)
		s.Require().NotNil(grant)
		mintAuthz, ok := grant.(*ethiqtypes.MintHaqqAuthorization)
		s.Require().True(ok)
		s.Require().Equal(huge.String(), mintAuthz.SpendLimit.Amount.String())
	})

	s.Run("increaseAllowance accepts a sum that still fits", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		_, err := s.precompile.Approve(ctx, granter.Addr, s.network.GetStateDB(), &approveMethod,
			[]interface{}{grantee, big.NewInt(100), methods})
		s.Require().NoError(err)

		_, err = s.precompile.IncreaseAllowance(ctx, granter.Addr, s.network.GetStateDB(), &increaseMethod,
			[]interface{}{grantee, big.NewInt(50), methods})
		s.Require().NoError(err)

		grant, _ := s.network.App.AuthzKeeper.GetAuthorization(
			ctx,
			sdk.AccAddress(grantee.Bytes()),
			granter.AccAddr,
			ethiq.MintHaqqMsgURL,
		)
		mintAuthz, ok := grant.(*ethiqtypes.MintHaqqAuthorization)
		s.Require().True(ok)
		s.Require().Equal("150", mintAuthz.SpendLimit.Amount.String())
	})
}

// TestRevokeDropsWholeApplicationGrant covers the single call that cuts a grantee off.
//
// revokeApplicationID removes one ID at a time, which is useless for incident response: the
// granter cannot even enumerate the approved IDs on chain. revoke deletes the grant outright,
// the way the staking precompile treats its own message types.
func (s *PrecompileTestSuite) TestRevokeDropsWholeApplicationGrant() {
	granter := s.keyring.GetAddr(0)
	grantee := s.keyring.GetAddr(1)
	appMethods := []string{ethiq.MsgMintHaqqByApplicationMsgURL}

	approveMethod := s.precompile.Methods[ethiq.ApproveApplicationIDMethod]
	revokeMethod := s.precompile.Methods[authorization.RevokeMethod]

	grantOf := func(ctx sdk.Context, msgURL string) authztypes.Authorization {
		auth, _ := s.network.App.AuthzKeeper.GetAuthorization(
			ctx,
			sdk.AccAddress(grantee.Bytes()),
			sdk.AccAddress(granter.Bytes()),
			msgURL,
		)
		return auth
	}

	s.Run("deletes a grant covering several application IDs", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		for _, appID := range []int64{0, 1, 2} {
			_, err := s.precompile.ApproveApplicationID(
				ctx, granter, s.network.GetStateDB(), &approveMethod,
				[]interface{}{grantee, big.NewInt(appID), appMethods},
			)
			s.Require().NoError(err)
		}
		s.Require().NotNil(grantOf(ctx, ethiq.MsgMintHaqqByApplicationMsgURL))

		stateDB := s.network.GetStateDB()
		bz, err := s.precompile.Revoke(ctx, granter, stateDB, &revokeMethod,
			[]interface{}{grantee, appMethods})
		s.Require().NoError(err)
		s.Require().True(unpackBool(s, authorization.RevokeMethod, bz))

		s.Require().Nil(grantOf(ctx, ethiq.MsgMintHaqqByApplicationMsgURL))
		s.Require().Equal([]string{authorization.EventTypeRevocation}, s.eventNames(stateDB.Logs()),
			"a full revoke is reported by Revocation alone")
	})

	s.Run("clears a grant of a foreign authorization type", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		// A GenericAuthorization under the same URL can only arrive through a Cosmos MsgGrant.
		// approveApplicationID and revokeApplicationID both refuse it on the type assertion,
		// so revoke is the only way to clear it from the EVM side.
		expiration := ctx.BlockTime().Add(time.Hour)
		s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
			ctx,
			sdk.AccAddress(grantee.Bytes()),
			sdk.AccAddress(granter.Bytes()),
			authztypes.NewGenericAuthorization(ethiq.MsgMintHaqqByApplicationMsgURL),
			&expiration,
		))

		revokeAppIDMethod := s.precompile.Methods[ethiq.RevokeApplicationIDMethod]
		_, err := s.precompile.RevokeApplicationID(
			ctx, granter, s.network.GetStateDB(), &revokeAppIDMethod,
			[]interface{}{grantee, big.NewInt(0), appMethods},
		)
		s.Require().ErrorContains(err, "expected: *ethiqtypes.MintHaqqByApplicationIDAuthorization")

		_, err = s.precompile.Revoke(ctx, granter, s.network.GetStateDB(), &revokeMethod,
			[]interface{}{grantee, appMethods})
		s.Require().NoError(err)
		s.Require().Nil(grantOf(ctx, ethiq.MsgMintHaqqByApplicationMsgURL))
	})

	s.Run("drops both message types in one call", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		approveMintMethod := s.precompile.Methods[authorization.ApproveMethod]
		_, err := s.precompile.Approve(ctx, granter, s.network.GetStateDB(), &approveMintMethod,
			[]interface{}{grantee, big.NewInt(100), []string{ethiq.MintHaqqMsgURL}})
		s.Require().NoError(err)
		_, err = s.precompile.ApproveApplicationID(
			ctx, granter, s.network.GetStateDB(), &approveMethod,
			[]interface{}{grantee, big.NewInt(0), appMethods},
		)
		s.Require().NoError(err)

		_, err = s.precompile.Revoke(ctx, granter, s.network.GetStateDB(), &revokeMethod,
			[]interface{}{grantee, []string{ethiq.MintHaqqMsgURL, ethiq.MsgMintHaqqByApplicationMsgURL}})
		s.Require().NoError(err)

		s.Require().Nil(grantOf(ctx, ethiq.MintHaqqMsgURL))
		s.Require().Nil(grantOf(ctx, ethiq.MsgMintHaqqByApplicationMsgURL))
	})

	s.Run("fails when there is no such grant", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		_, err := s.precompile.Revoke(ctx, granter, s.network.GetStateDB(), &revokeMethod,
			[]interface{}{grantee, appMethods})
		s.Require().ErrorContains(err, "authorization not found")
	})

	s.Run("reverts the whole call when one type URL has no grant", func() {
		s.SetupTest()
		ctx := s.network.GetContext()

		_, err := s.precompile.ApproveApplicationID(
			ctx, granter, s.network.GetStateDB(), &approveMethod,
			[]interface{}{grantee, big.NewInt(0), appMethods},
		)
		s.Require().NoError(err)

		// MintHaqq was never approved; the precompile call is atomic, so nothing is revoked.
		_, err = s.precompile.Revoke(ctx, granter, s.network.GetStateDB(), &revokeMethod,
			[]interface{}{grantee, []string{ethiq.MintHaqqMsgURL, ethiq.MsgMintHaqqByApplicationMsgURL}})
		s.Require().Error(err)
	})
}

// TestApproveApplicationIDRejectsUnusableApplications covers what the tightened
// MintHaqqByApplicationIDAuthorization.ValidateBasic means at the precompile surface: a grant
// that could never be executed now fails at approval rather than much later at the mint.
func (s *PrecompileTestSuite) TestApproveApplicationIDRejectsUnusableApplications() {
	s.SetupTest()
	ctx := s.network.GetContext()
	granter := s.keyring.GetAddr(0)
	grantee := s.keyring.GetAddr(1)
	methods := []string{ethiq.MsgMintHaqqByApplicationMsgURL}
	method := s.precompile.Methods[ethiq.ApproveApplicationIDMethod]

	var canceled uint64
	found := false
	for id := uint64(0); id < ethiqtypes.TotalNumberOfApplications(); id++ {
		if ethiqtypes.IsApplicationCanceled(id) {
			canceled, found = id, true
			break
		}
	}
	s.Require().True(found, "no canceled application in the waitlist")

	_, err := s.precompile.ApproveApplicationID(
		ctx, granter, s.network.GetStateDB(), &method,
		[]interface{}{grantee, new(big.Int).SetUint64(canceled), methods},
	)
	s.Require().ErrorContains(err, "is canceled")

	grant, _ := s.network.App.AuthzKeeper.GetAuthorization(
		ctx,
		sdk.AccAddress(grantee.Bytes()),
		sdk.AccAddress(granter.Bytes()),
		ethiq.MsgMintHaqqByApplicationMsgURL,
	)
	s.Require().Nil(grant, "a rejected approval must not leave a grant behind")

	_, err = s.precompile.ApproveApplicationID(
		ctx, granter, s.network.GetStateDB(), &method,
		[]interface{}{grantee, new(big.Int).SetUint64(ethiqtypes.TotalNumberOfApplications()), methods},
	)
	s.Require().ErrorContains(err, "does not exist")
}

// TestAllowancePointsAtTheAuthzQuery checks the error a caller gets for asking allowance about
// an application grant. There is no scalar allowance to return, and the answer lives off chain
// in the standard authz Grants query, so the message has to say so instead of reporting a bare
// type mismatch.
func (s *PrecompileTestSuite) TestAllowancePointsAtTheAuthzQuery() {
	s.SetupTest()
	ctx := s.network.GetContext()
	granter := s.keyring.GetAddr(0)
	grantee := s.keyring.GetAddr(1)

	allowanceMethod := s.precompile.Methods[authorization.AllowanceMethod]
	approveMethod := s.precompile.Methods[ethiq.ApproveApplicationIDMethod]

	// No grant at all: allowance answers zero rather than failing.
	bz, err := s.precompile.Allowance(ctx, &allowanceMethod, nil,
		[]interface{}{grantee, granter, ethiq.MsgMintHaqqByApplicationMsgURL})
	s.Require().NoError(err)
	values, err := allowanceMethod.Outputs.Unpack(bz)
	s.Require().NoError(err)
	remaining, ok := values[0].(*big.Int)
	s.Require().True(ok)
	s.Require().Zero(remaining.Sign())

	_, err = s.precompile.ApproveApplicationID(
		ctx, granter, s.network.GetStateDB(), &approveMethod,
		[]interface{}{grantee, big.NewInt(0), []string{ethiq.MsgMintHaqqByApplicationMsgURL}},
	)
	s.Require().NoError(err)

	_, err = s.precompile.Allowance(ctx, &allowanceMethod, nil,
		[]interface{}{grantee, granter, ethiq.MsgMintHaqqByApplicationMsgURL})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "cosmos.authz.v1beta1.Query/Grants")
	s.Require().ErrorContains(err, ethiq.MsgMintHaqqByApplicationMsgURL)
}
