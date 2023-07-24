package v150_test

import (
	"math/big"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/types"

	"github.com/haqq-network/haqq/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	utils "github.com/haqq-network/haqq/types"
)

type testAcc struct {
	accPrivateKey *ethsecp256k1.PrivKey
	ethAddress    common.Address
	accAddress    sdk.AccAddress
	signer        keyring.Signer
}

const denomExp = uint64(10e17)

var _ = Describe("Performing EVM contract calls", Ordered, func() {
	BeforeEach(func() {
		s.SetupTest()

		s.accounts = make([]testAcc, 10)
		for i := 0; i < 10; i++ {
			s.accounts[i].accPrivateKey, _ = ethsecp256k1.GenerateKey()
			s.accounts[i].ethAddress = common.BytesToAddress(s.accounts[i].accPrivateKey.PubKey().Address().Bytes())
			s.accounts[i].accAddress = sdk.AccAddress(s.accounts[i].ethAddress.Bytes())
			s.accounts[i].signer = utiltx.NewSigner(s.accounts[i].accPrivateKey)
		}

		hundredISLM := sdk.NewCoin(utils.BaseDenom, sdk.NewIntFromUint64(denomExp))
		hundredISLM.Amount = hundredISLM.Amount.MulRaw(100)
		err := testutil.FundAccount(s.ctx, s.app.BankKeeper, s.accounts[0].accAddress, sdk.NewCoins(hundredISLM))
		Expect(err).To(BeNil())
		s.Commit()
	})

	Context("Before conversion into VestingAccount", func() {
		It("success - deposit and withdraw tokens", func() {
			// check contract account type
			contractAcc := s.app.AccountKeeper.GetAccount(s.ctx, s.contractAddress.Bytes())
			_, ok := contractAcc.(*types.EthAccount)
			Expect(ok).To(BeTrue())

			// check contract balance before deposit
			balance := s.app.BankKeeper.GetBalance(s.ctx, sdk.AccAddress(s.contractAddress.Bytes()), utils.BaseDenom)
			Expect(balance.IsZero()).To(BeTrue())

			// deposit contract
			amount := big.NewInt(0)
			amount.SetUint64(denomExp)
			rsp, err := depositContract(s.accounts[0], s.accounts[1], amount)
			Expect(err).To(BeNil())
			Expect(rsp.VmError).To(BeEmpty())

			// check balance after deposit
			balanceAfter := s.app.BankKeeper.GetBalance(s.ctx, sdk.AccAddress(s.contractAddress.Bytes()), utils.BaseDenom)
			Expect(balanceAfter.IsZero()).To(BeFalse())
			Expect(balanceAfter.Equal(sdk.NewCoin(utils.BaseDenom, sdk.NewIntFromUint64(denomExp)))).To(BeTrue())

			rsp2, err := withdrawContract(s.accounts[1], amount)
			Expect(err).To(BeNil())
			Expect(rsp2.VmError).To(BeEmpty())

			balanceFinal := s.app.BankKeeper.GetBalance(s.ctx, sdk.AccAddress(s.contractAddress.Bytes()), utils.BaseDenom)
			Expect(balanceFinal.IsZero()).To(BeTrue())
			// check balances after withdrawal
			oneISLM := sdk.NewCoin(utils.BaseDenom, sdk.NewIntFromUint64(denomExp))
			balanceBenefeciary := s.app.BankKeeper.GetBalance(s.ctx, s.accounts[1].accAddress, utils.BaseDenom)
			Expect(balanceBenefeciary.IsZero()).To(BeFalse())
			Expect(balanceBenefeciary.IsGTE(oneISLM)).To(BeTrue())
		})
	})
})
