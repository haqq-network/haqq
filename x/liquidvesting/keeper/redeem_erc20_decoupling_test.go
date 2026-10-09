package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
)

// disableTokenPair flips the liquid denom's ERC20 pair off, the way a governance
// MsgToggleConversion would.
func (suite *KeeperTestSuite) disableTokenPair(ctx sdk.Context, denom string) {
	id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, denom)
	suite.Require().NotEmpty(id)
	pair, found := suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
	suite.Require().True(found)
	pair.Enabled = false
	suite.network.App.Erc20Keeper.SetTokenPair(ctx, pair)
}

func (suite *KeeperTestSuite) tokenPairEnabled(ctx sdk.Context, denom string) bool {
	id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, denom)
	suite.Require().NotEmpty(id)
	pair, found := suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
	suite.Require().True(found)
	return pair.Enabled
}

// TestRedeemWorksWithDisabledTokenPair covers the first coupling: redeeming burns the liquid
// denom and pays out the principal the module holds, which needs nothing from x/erc20. Gating it
// on pair.Enabled stranded the principal while the denom stayed transferable - the bank does not
// consult the flag, and neither does InstantiateERC20Precompile.
func (suite *KeeperTestSuite) TestRedeemWorksWithDisabledTokenPair() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	holder := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	suite.setupPartiallyLockedDenom(ctx)
	suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, holder))
	suite.Require().NoError(testutil.FundAccount(ctx, suite.network.App.BankKeeper, holder, liquidDenomAmount))

	suite.disableTokenPair(ctx, "aLIQUID0")

	before := suite.network.App.BankKeeper.GetBalance(ctx, holder, utils.BaseDenom)
	redeemed := sdk.NewInt64Coin("aLIQUID0", 1_000_000)

	err := suite.network.App.LiquidVestingKeeper.Redeem(ctx, holder, holder, redeemed)
	suite.Require().NoError(err, "a disabled ERC20 pair must not block redemption")

	after := suite.network.App.BankKeeper.GetBalance(ctx, holder, utils.BaseDenom)
	suite.Require().Equal(redeemed.Amount, after.Amount.Sub(before.Amount),
		"principal must be paid out in full")
}

// TestRedeemWorksWithErc20GloballyDisabled covers the second, asymmetric coupling: partial
// redemptions never touched x/erc20, but the final one - the one that zeroes the schedule -
// went through the erc20 msg server, which refuses to run while EnableErc20 is off. The last
// holders could not exit.
func (suite *KeeperTestSuite) TestRedeemWorksWithErc20GloballyDisabled() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	holder := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	suite.setupPartiallyLockedDenom(ctx)
	suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, holder))
	suite.Require().NoError(testutil.FundAccount(ctx, suite.network.App.BankKeeper, holder, liquidDenomAmount))

	erc20Params := suite.network.App.Erc20Keeper.GetParams(ctx)
	erc20Params.EnableErc20 = false
	suite.Require().NoError(suite.network.App.Erc20Keeper.SetParams(ctx, erc20Params))
	suite.Require().False(suite.network.App.Erc20Keeper.IsERC20Enabled(ctx))

	// Redeem the whole supply so the schedule zeroes out and the terminal branch runs.
	err := suite.network.App.LiquidVestingKeeper.Redeem(ctx, holder, holder, liquidDenomAmount[0])
	suite.Require().NoError(err, "the terminal redemption must not depend on the global erc20 flag")

	_, found := suite.network.App.LiquidVestingKeeper.GetDenom(ctx, "aLIQUID0")
	suite.Require().False(found, "the exhausted denom must be deleted")
	suite.Require().False(suite.tokenPairEnabled(ctx, "aLIQUID0"),
		"cleanup must still disable the pair of an exhausted denom")
}

// TestRedeemSurvivesMissingTokenPair pins down that a denom whose pair was never registered - a
// bookkeeping inconsistency - does not cost the holder their principal.
func (suite *KeeperTestSuite) TestRedeemSurvivesMissingTokenPair() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	holder := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	suite.setupPartiallyLockedDenom(ctx)
	suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, holder))
	suite.Require().NoError(testutil.FundAccount(ctx, suite.network.App.BankKeeper, holder, liquidDenomAmount))

	// Drop the denom → pair mapping, leaving the liquid denom without an ERC20 representation.
	suite.network.App.Erc20Keeper.SetDenomMap(ctx, "aLIQUID0", []byte{})

	err := suite.network.App.LiquidVestingKeeper.Redeem(ctx, holder, holder, sdk.NewInt64Coin("aLIQUID0", 1_000_000))
	suite.Require().NoError(err, "a missing ERC20 pair must not block redemption")
}
