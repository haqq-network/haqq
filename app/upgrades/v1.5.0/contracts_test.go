package v150_test

import (
	"encoding/json"
	"math/big"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/server/config"
	evm "github.com/evmos/ethermint/x/evm/types"

	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/contracts"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
)

type testAcc struct {
	accPrivateKey *ethsecp256k1.PrivKey
	ethAddress    common.Address
	accAddress    sdk.AccAddress
	signer        keyring.Signer
}

const oneISLM = uint64(10e17)

var _ = Describe("Performing EVM contract calls before revesting", Ordered, func() {
	BeforeEach(func() {
		s.SetupTest()
	})

	bondDenom := s.app.StakingKeeper.BondDenom(s.ctx)
	var err error
	accounts := make([]testAcc, 10)
	for i := 0; i < 10; i++ {
		accounts[i].accPrivateKey, err = ethsecp256k1.GenerateKey()
		Expect(err).To(BeNil())
		accounts[i].ethAddress = common.BytesToAddress(accounts[i].accPrivateKey.PubKey().Address().Bytes())
		accounts[i].accAddress = sdk.AccAddress(accounts[i].ethAddress.Bytes())
		accounts[i].signer = utiltx.NewSigner(accounts[i].accPrivateKey)
	}

	Context("deposit contract", func() {
		It("should be successful", func() {
			tenISLM := sdk.NewCoin(bondDenom, sdk.NewIntFromUint64(10*oneISLM))
			err = testutil.FundAccount(s.ctx, s.app.BankKeeper, accounts[0].accAddress, sdk.NewCoins(tenISLM))
			Expect(err).To(BeNil())
			s.Commit()

			// deposit contract
			callData, err := contracts.HaqqTestingContract.ABI.Pack("deposit", accounts[1].ethAddress)
			Expect(err).To(BeNil())

			amount := big.NewInt(0)
			amount.SetUint64(oneISLM)
			hexAmount := hexutil.Big(*amount)

			args, err := json.Marshal(&evm.TransactionArgs{
				To:    &s.contractAddress,
				From:  &accounts[0].ethAddress,
				Value: &hexAmount,
				Data:  (*hexutil.Bytes)(&callData),
			})
			Expect(err).To(BeNil())

			res, err := s.queryClientEvm.EstimateGas(s.ctx, &evm.EthCallRequest{
				Args:   args,
				GasCap: config.DefaultGasCap,
			})
			Expect(err).To(BeNil())

			// Mint the max gas to the FeeCollector to ensure balance in case of refund
			s.MintFeeCollector(sdk.NewCoins(sdk.NewCoin(evm.DefaultEVMDenom, sdk.NewInt(s.app.FeeMarketKeeper.GetBaseFee(s.ctx).Int64()*int64(res.Gas)))))

			nonce := s.app.EvmKeeper.GetNonce(s.ctx, accounts[0].ethAddress)

			ercTransferTx := evm.NewTx(
				s.app.EvmKeeper.ChainID(),
				nonce,
				&s.contractAddress,
				amount,
				res.Gas,
				nil,
				s.app.FeeMarketKeeper.GetBaseFee(s.ctx),
				big.NewInt(1),
				callData,
				&ethtypes.AccessList{}, // accesses
			)

			ercTransferTx.From = accounts[0].ethAddress.Hex()
			err = ercTransferTx.Sign(ethtypes.LatestSignerForChainID(s.app.EvmKeeper.ChainID()), accounts[0].signer)
			Expect(err).To(BeNil())
			rsp, err := s.app.EvmKeeper.EthereumTx(s.ctx, ercTransferTx)
			Expect(err).To(BeNil())
			Expect(rsp.VmError).To(BeEmpty())
		})
	})
})
