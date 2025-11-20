package keeper_test

import (
	. "github.com/onsi/gomega"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/x/vesting/types"
)

func (suite *KeeperTestSuite) setupClawbackVestingAccount(vestingAccount, funder keyring.Key, vestingPeriods, lockupPeriods sdkvesting.Periods) *types.ClawbackVestingAccount {
	// Create and fund the clawback vesting accounts
	vestingStart := suite.network.GetContext().BlockTime()

	// create a clawback vesting account
	msgConv := types.NewMsgConvertIntoVestingAccount(
		funder.AccAddr,
		vestingAccount.AccAddr,
		vestingStart,
		lockupPeriods,
		vestingPeriods,
		true, false, nil,
	)
	res, err := suite.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msgConv}})
	Expect(err).To(BeNil())
	Expect(res.IsOK()).To(BeTrue())
	Expect(suite.network.NextBlock()).To(BeNil())

	acc, err := suite.handler.GetAccount(vestingAccount.AccAddr.String())
	Expect(err).To(BeNil())
	var ok bool
	clawbackAccount, ok := acc.(*types.ClawbackVestingAccount)
	Expect(ok).To(BeTrue())

	return clawbackAccount
}
