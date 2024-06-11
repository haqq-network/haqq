package keeper_test

import (
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/x/vesting/types"
)

func (s *KeeperIntegrationTestSuite) setupClawbackVestingAccount(vestingAccount, funder keyring.Key, vestingPeriods, lockupPeriods sdkvesting.Periods) *types.ClawbackVestingAccount {
	vestingStart := s.network.GetContext().BlockTime()

	// send a create vesting account tx
	createAccMsg := types.NewMsgConvertIntoVestingAccount(
		funder.AccAddr,
		vestingAccount.AccAddr,
		vestingStart,
		lockupPeriods,
		vestingPeriods,
		false,
		false,
		nil,
	)
	res, err := s.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{createAccMsg}, Gas: &gasLimit, GasPrice: &gasPrice})
	Expect(err).To(BeNil())
	Expect(res.IsOK()).To(BeTrue())
	Expect(s.network.NextBlock()).To(BeNil())

	// Check the clawback vesting account exists
	acc, err := s.grpcHandler.GetAccount(vestingAccount.AccAddr.String())
	Expect(err).To(BeNil())
	clawbackAccount, ok := acc.(*types.ClawbackVestingAccount)
	Expect(ok).To(BeTrue())

	return clawbackAccount
}
