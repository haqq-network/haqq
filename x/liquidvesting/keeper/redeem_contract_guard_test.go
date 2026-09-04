package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/haqq-network/haqq/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/utils"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// setupPartiallyLockedDenom registers aLIQUID0 with a schedule whose tail is still in the
// future, so Redeem reaches ApplyVestingSchedule instead of handing out unlocked coins.
func (suite *KeeperTestSuite) setupPartiallyLockedDenom(ctx sdk.Context) {
	err := testutil.FundModuleAccount(ctx, suite.network.App.BankKeeper, types.ModuleName, amount)
	suite.Require().NoError(err)

	// half way into the second of three periods
	startTime := ctx.BlockTime().Add(-150000 * time.Second)
	suite.network.App.LiquidVestingKeeper.SetDenom(ctx, types.Denom{
		BaseDenom:     "aLIQUID0",
		DisplayDenom:  "LIQUID0",
		OriginalDenom: utils.BaseDenom,
		StartTime:     startTime,
		EndTime:       startTime.Add(lockupPeriods.TotalDuration()),
		LockupPeriods: lockupPeriods,
	})

	md := banktypes.Metadata{
		Description: "Liquid vesting token",
		DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
		Base:        "aLIQUID0",
		Display:     "LIQUID0",
		Name:        "LIQUID0",
		Symbol:      "LIQUID0",
	}
	suite.network.App.BankKeeper.SetDenomMetaData(ctx, md)

	fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, md.Base, utils.BaseDenom)
	tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
	suite.Require().NoError(err)
	tokenPair.Denom = md.Base
	suite.network.App.Erc20Keeper.SetTokenPair(ctx, tokenPair)
	suite.network.App.Erc20Keeper.SetDenomMap(ctx, tokenPair.Denom, tokenPair.GetID())
	suite.network.App.Erc20Keeper.SetERC20Map(ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())
	suite.Require().NoError(suite.network.App.Erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract()))
}

// newContractAccount stores an EthAccount carrying a non-empty code hash, i.e. a contract.
func (suite *KeeperTestSuite) newContractAccount(ctx sdk.Context, addr sdk.AccAddress) *haqqtypes.EthAccount {
	base := suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, addr).(*haqqtypes.EthAccount)
	suite.Require().NoError(base.SetCodeHash(common.BytesToHash(crypto.Keccak256([]byte{0x60, 0x00}))))
	suite.network.App.AccountKeeper.SetAccount(ctx, base)
	return base
}

// TestRedeemCannotConvertForeignContract covers the third-party half of the guard: redeemTo is
// unconstrained, so without the check anyone could push a clawback vesting schedule onto someone
// else's contract for the price of a dust redeem - and a contract can never undo it, since
// MsgConvertVestingAccount is signed by the account itself and the vesting precompile is not
// registered.
func (suite *KeeperTestSuite) TestRedeemCannotConvertForeignContract() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	redeemer := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	victim := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	suite.setupPartiallyLockedDenom(ctx)
	suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, redeemer))
	victimAcc := suite.newContractAccount(ctx, victim)
	suite.Require().NoError(testutil.FundAccount(ctx, suite.network.App.BankKeeper, redeemer, liquidDenomAmount))

	err := suite.network.App.LiquidVestingKeeper.Redeem(
		ctx, redeemer, victim, sdk.NewInt64Coin("aLIQUID0", 1_000_000),
	)
	suite.Require().Error(err, "a third party must not convert someone else's contract")
	suite.Require().Contains(err.Error(), "is a contract account and cannot be converted into a clawback vesting account by another account")

	after := suite.network.App.AccountKeeper.GetAccount(ctx, victim)
	_, converted := after.(*vestingtypes.ClawbackVestingAccount)
	suite.Require().False(converted, "victim contract must not have been converted")
	ethAcc, ok := after.(*haqqtypes.EthAccount)
	suite.Require().True(ok, "victim must still be an EthAccount")
	suite.Require().Equal(victimAcc.GetCodeHash(), ethAcc.GetCodeHash(), "code hash must be untouched")
}

// TestRedeemAllowsContractSelfRedeem covers the other half: x/ethiq redeems a Safe's own liquid
// balance back into it (redeemAllLiquidVestingCoins passes redeemFrom == redeemTo), so the guard
// must not reject a contract acting on itself through an authorized call.
func (suite *KeeperTestSuite) TestRedeemAllowsContractSelfRedeem() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	contractAddr := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	suite.setupPartiallyLockedDenom(ctx)
	suite.newContractAccount(ctx, contractAddr)
	suite.Require().NoError(testutil.FundAccount(ctx, suite.network.App.BankKeeper, contractAddr, liquidDenomAmount))

	err := suite.network.App.LiquidVestingKeeper.Redeem(
		ctx, contractAddr, contractAddr, sdk.NewInt64Coin("aLIQUID0", 1_000_000),
	)
	suite.Require().NoError(err, "self-redeem must keep working for contracts - x/ethiq depends on it")
}
