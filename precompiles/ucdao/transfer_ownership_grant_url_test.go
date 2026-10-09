package ucdao_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestTransferOwnershipGrantRoundTrip pins the fix for the grant/URL mismatch.
//
// authzkeeper.SaveGrant keys a grant by authorization.MsgTypeURL(), while DeleteGrant and
// GetAuthorization use the URL the caller passes. TransferOwnershipAuthorization reports
// MsgTransferOwnershipWithAmount - and its Accept casts the message to
// *MsgTransferOwnershipWithAmount, so that is also the key cosmos authz uses on the
// MsgExec path. The precompile used to name MsgTransferOwnership instead, so approve()
// wrote a grant that revoke(), allowance() and the allowance deltas could never find:
// once created from the EVM, it could not be revoked from the EVM at all.
//
// The invariant this test locks in: whatever URL approve() accepts, revoke() must be able
// to remove and allowance() must be able to read.
func (s *PrecompileTestSuite) TestTransferOwnershipGrantRoundTrip() {
	s.SetupTest()
	ctx := s.network.GetContext()
	stDB := s.network.GetStateDB()

	granter := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)

	url := ucdao.TransferOwnershipWithAmountMsgURL
	s.Require().Equal(
		(&ucdaotypes.TransferOwnershipAuthorization{}).MsgTypeURL(), url,
		"the precompile must name the URL the authorization actually stores itself under",
	)

	approveMethod := s.precompile.Methods[authorization.ApproveMethod]
	allowanceMethod := s.precompile.Methods[authorization.AllowanceMethod]
	revokeMethod := s.precompile.Methods[authorization.RevokeMethod]

	readAllowance := func() *big.Int {
		bz, err := s.precompile.Allowance(ctx, &allowanceMethod, nil, []interface{}{
			grantee.Addr, granter.Addr, url,
		})
		s.Require().NoError(err)
		out, err := allowanceMethod.Outputs.Unpack(bz)
		s.Require().NoError(err)
		return out[0].(*big.Int)
	}

	// ------------------------------------------------------------------- approve
	_, err := s.precompile.Approve(ctx, granter.Addr, stDB, &approveMethod, []interface{}{
		grantee.Addr, big.NewInt(1e18), []string{url},
	})
	s.Require().NoError(err)

	grant, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee.AccAddr, granter.AccAddr, url)
	s.Require().NotNil(grant, "the grant must be readable under the URL that was approved")
	transferAuthz, ok := grant.(*ucdaotypes.TransferOwnershipAuthorization)
	s.Require().True(ok)
	s.Require().Equal(big.NewInt(1e18), transferAuthz.SpendLimit.Amount.BigInt())
	s.Require().Zero(readAllowance().Cmp(big.NewInt(1e18)), "allowance() must report it")

	// -------------------------------------------------------- increaseAllowance
	increaseMethod := s.precompile.Methods[authorization.IncreaseAllowanceMethod]
	_, err = s.precompile.IncreaseAllowance(ctx, granter.Addr, stDB, &increaseMethod, []interface{}{
		grantee.Addr, big.NewInt(1e18), []string{url},
	})
	s.Require().NoError(err)
	s.Require().Zero(readAllowance().Cmp(big.NewInt(2e18)))

	// -------------------------------------------------------- decreaseAllowance
	decreaseMethod := s.precompile.Methods[authorization.DecreaseAllowanceMethod]
	_, err = s.precompile.DecreaseAllowance(ctx, granter.Addr, stDB, &decreaseMethod, []interface{}{
		grantee.Addr, big.NewInt(5e17), []string{url},
	})
	s.Require().NoError(err)
	s.Require().Zero(readAllowance().Cmp(big.NewInt(15e17)))

	// -------------------------------------------------------------------- revoke
	_, err = s.precompile.Revoke(ctx, granter.Addr, stDB, &revokeMethod, []interface{}{
		grantee.Addr, []string{url},
	})
	s.Require().NoError(err, "a grant created from the EVM must be revocable from the EVM")

	gone, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee.AccAddr, granter.AccAddr, url)
	s.Require().Nil(gone)
	s.Require().Zero(readAllowance().Sign())
}

// TestTransferOwnershipURLIsRejectedWithAHint documents the deliberate consequence of the
// fix: MsgTransferOwnership is no longer accepted by the authorization methods. It
// carries no amount, so a spend limit cannot be expressed for it and ucDAO registers no
// authorization type for it. The error has to say that rather than read as "unknown
// message type", because it is the URL callers are most likely to try.
func (s *PrecompileTestSuite) TestTransferOwnershipURLIsRejectedWithAHint() {
	s.SetupTest()
	ctx := s.network.GetContext()
	stDB := s.network.GetStateDB()

	granter := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)
	legacyURL := sdk.MsgTypeURL(&ucdaotypes.MsgTransferOwnership{})

	for _, tc := range []struct {
		name string
		call func(args []interface{}) error
		args []interface{}
	}{
		{
			authorization.ApproveMethod,
			func(args []interface{}) error {
				m := s.precompile.Methods[authorization.ApproveMethod]
				_, err := s.precompile.Approve(ctx, granter.Addr, stDB, &m, args)
				return err
			},
			[]interface{}{grantee.Addr, big.NewInt(1e18), []string{legacyURL}},
		},
		{
			authorization.RevokeMethod,
			func(args []interface{}) error {
				m := s.precompile.Methods[authorization.RevokeMethod]
				_, err := s.precompile.Revoke(ctx, granter.Addr, stDB, &m, args)
				return err
			},
			[]interface{}{grantee.Addr, []string{legacyURL}},
		},
		{
			authorization.IncreaseAllowanceMethod,
			func(args []interface{}) error {
				m := s.precompile.Methods[authorization.IncreaseAllowanceMethod]
				_, err := s.precompile.IncreaseAllowance(ctx, granter.Addr, stDB, &m, args)
				return err
			},
			[]interface{}{grantee.Addr, big.NewInt(1), []string{legacyURL}},
		},
		{
			authorization.DecreaseAllowanceMethod,
			func(args []interface{}) error {
				m := s.precompile.Methods[authorization.DecreaseAllowanceMethod]
				_, err := s.precompile.DecreaseAllowance(ctx, granter.Addr, stDB, &m, args)
				return err
			},
			[]interface{}{grantee.Addr, big.NewInt(1), []string{legacyURL}},
		},
	} {
		s.Run(tc.name, func() {
			err := tc.call(tc.args)
			s.Require().Error(err)
			s.Require().ErrorContains(err, "has no ucdao authorization type")
			s.Require().ErrorContains(err, ucdao.TransferOwnershipWithAmountMsgURL)
		})
	}
}

// TestTransferOwnershipRejectsContractCallers is the other half of the same consequence:
// the full-balance transferOwnership method cannot be delegated, so a contract caller is
// turned away with an explicit reason rather than a misleading "grant does not exist".
func (s *PrecompileTestSuite) TestTransferOwnershipRejectsContractCallers() {
	s.SetupTest()
	ctx := s.network.GetContext()
	stDB := s.network.GetStateDB()

	origin := s.keyring.GetAddr(0)
	callerContract := s.keyring.GetAddr(1) // stands in for a contract caller: caller != origin
	newOwner := s.keyring.GetAddr(1)

	method := s.precompile.Methods[ucdao.TransferOwnershipMethod]
	contract := vm.NewPrecompile(vm.AccountRef(callerContract), s.precompile, big.NewInt(0), uint64(1e6))

	_, err := s.precompile.TransferOwnership(ctx, origin, contract, stDB, &method,
		[]interface{}{origin, newOwner})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "transferOwnership cannot be called on behalf of another account")
	s.Require().ErrorContains(err, ucdao.TransferOwnershipWithAmountMsgURL)
}
