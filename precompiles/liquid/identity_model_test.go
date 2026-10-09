package liquid_test

import (
	"math/big"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthz "github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/haqq-network/haqq/precompiles/liquid"
	"github.com/haqq-network/haqq/precompiles/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	liquidtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
)

// Regression tests for the precompile identity model in liquid.
//
// liquid exposes no approve, so it has no self-grant hole to close. What it shares
// with the other precompiles is the consuming side: the granter is the sender
// carried by the message -- the account whose vesting position moves -- and not
// tx.origin.
//
// The gate `!isCallerSender && origin != sender` leaves exactly three reachable
// branches, so the change from origin to sender is observable in one of them only:
//
//	A  caller == origin == sender   direct EOA call, no grant, unchanged
//	B1 caller == sender != origin   contract acting on its OWN position
//	B2 caller != sender == origin   third party, grant from the sender, unchanged
//
// B1 is the branch these tests exist for. Before the fix it demanded a grant from
// an unrelated origin to move the caller's own funds, which is the contract-wallet
// defect: a Safe could not touch its own vesting position until some executor EOA
// granted it. Nothing else in the package distinguishes the two models -- every
// other liquid test has sender == origin, where the old and the new granter are the
// same account -- so without these, reverting the fix passes the suite.

// TestContractLiquidatesOwnPositionWithoutGrant is branch B1: the caller is the
// sender, so its own code is the authorization and no grant of any kind exists.
func (s *PrecompileTestSuite) TestContractLiquidatesOwnPositionWithoutGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	// The contract owns the vesting position it liquidates.
	contractAddr := utiltx.GenerateAddress()
	contractAccAddr := sdk.AccAddress(contractAddr.Bytes())
	s.createClawbackVestingAccount(ctx, contractAccAddr)

	// An unrelated EOA signs the transaction, so origin != caller.
	originAddr := utiltx.GenerateAddress()
	receiver := s.keyring.GetAddr(1)

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, contractAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, originAddr, contract, s.network.GetStateDB(), &method, []any{
		contractAddr,
		receiver,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err, "a contract must be able to liquidate its own position without a third party's grant")

	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	balance := s.network.App.BankKeeper.GetBalance(ctx, sdk.AccAddress(receiver.Bytes()), liquidDenom)
	s.Require().Equal(sdkmath.NewInt(1_000_000), balance.Amount,
		"the liquid tokens must come from the caller's own position")

	// And no grant was invented along the way in either direction.
	msgURL := sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{})
	grantFromOrigin, _ := s.network.App.AuthzKeeper.GetAuthorization(
		ctx, contractAccAddr, sdk.AccAddress(originAddr.Bytes()), msgURL,
	)
	s.Require().Nil(grantFromOrigin, "acting on its own position must neither need nor create a grant")
}

// TestContractRedeemsOwnPositionWithoutGrant is branch B1 for Redeem. Redeem is the
// worse of the two to gate on the wrong account: it burns every liquid token held
// and routes the principal onward.
func (s *PrecompileTestSuite) TestContractRedeemsOwnPositionWithoutGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	contractAddr := utiltx.GenerateAddress()
	contractAccAddr := sdk.AccAddress(contractAddr.Bytes())
	s.createClawbackVestingAccount(ctx, contractAccAddr)

	// Mint the liquid tokens onto the contract through the plain EOA-shaped path,
	// so the redeem below is the only step under test.
	liquidateMethod := s.precompile.Methods[liquid.LiquidateMethod]
	liqContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, contractAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, contractAddr, liqContract, s.network.GetStateDB(), &liquidateMethod, []any{
		contractAddr,
		contractAddr,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err)

	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	s.Require().Equal(
		sdkmath.NewInt(1_000_000),
		s.network.App.BankKeeper.GetBalance(ctx, contractAccAddr, liquidDenom).Amount,
	)

	// An unrelated EOA signs the transaction, so origin != caller.
	originAddr := utiltx.GenerateAddress()

	redeemMethod := s.precompile.Methods[liquid.RedeemMethod]
	redeemContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, contractAddr, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, originAddr, redeemContract, s.network.GetStateDB(), &redeemMethod, []any{
		contractAddr,
		contractAddr,
		liquidDenom,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err, "a contract must be able to redeem its own liquid tokens without a third party's grant")

	s.Require().True(
		s.network.App.BankKeeper.GetBalance(ctx, contractAccAddr, liquidDenom).Amount.IsZero(),
		"the caller's own liquid tokens must have been burned",
	)
}

// TestLiquidateForForeignAccountStillRejected pins the branch that is deliberately
// left unchanged: a caller may not act for an account that did not sign the
// transaction, even holding a grant from the transaction signer. A grant from
// tx.origin is not an authorization over a third party's position.
func (s *PrecompileTestSuite) TestLiquidateForForeignAccountStillRejected() {
	s.SetupTest()
	ctx := s.network.GetContext()

	// The position belongs to a third party: neither the caller nor the signer.
	victimAddr := utiltx.GenerateAddress()
	s.createClawbackVestingAccount(ctx, sdk.AccAddress(victimAddr.Bytes()))

	callerAddr := utiltx.GenerateAddress()
	callerAccAddr := sdk.AccAddress(callerAddr.Bytes())
	originAddr := utiltx.GenerateAddress()

	// The signer grants the caller everything it can grant. It still does not
	// reach the victim's position.
	expiration := ctx.BlockTime().Add(time.Hour)
	s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
		ctx, callerAccAddr, sdk.AccAddress(originAddr.Bytes()),
		sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{})),
		&expiration,
	))

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, originAddr, contract, s.network.GetStateDB(), &method, []any{
		victimAddr,
		s.keyring.GetAddr(1),
		big.NewInt(1_000_000),
	})
	s.Require().Error(err, "a grant from tx.origin must not reach a third party's position")
	s.Require().ErrorContains(err, "must match sender address")
}
