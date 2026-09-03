package ucdao_test

import (
	"fmt"
	"math/big"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	haqqtestutil "github.com/haqq-network/haqq/testutil"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

var (
	ucdaoFundAmount     = math.NewInt(100).Mul(math.NewInt(1e18)) // 100 aISLM in UC DAO
	ucdaoTransferAmount = math.NewInt(60).Mul(math.NewInt(1e18))  // mirrors partial transfer on mainnet
)

// TestLegacyUcdaoEscrowDoubleCreditViaContract reproduces the ucDAO precompile mint
// without ethiq / MsgConvertToHaqq.
//
// In one EVM tx a forwarder (1) sends 1 wei to the owner escrow (StateDB loads and dirties
// that account) and (2) calls transferOwnershipWithAmount on 0x0901. Cosmos SendCoins moves
// X aISLM to the newOwner escrow, but the precompile does not journal SubBalance/AddBalance.
// StateDB.Commit then SetBalance(ownerEscrow, stale B+1), minting X back. Result: owner still
// holds ~B, newOwner holds X, total escrow aISLM grows by X.
//
// ConvertToHaqq is only a later extraction step; the invariant break is already visible here.
func (s *PrecompileTestSuite) TestLegacyUcdaoEscrowDoubleCreditViaContract() {
	owner := s.keyring.GetKey(0)
	newOwner := s.keyring.GetKey(1)
	ownerEscrowEVM := common.BytesToAddress(ucdaotypes.GetEscrowAddress(owner.AccAddr).Bytes())
	ucdaoPrecompileAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)

	ctx := s.network.GetContext()
	bank := s.network.App.BankKeeper
	daoKeeper := s.network.App.DaoKeeper

	s.Require().NoError(
		haqqtestutil.FundAccount(ctx, bank, owner.AccAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, ucdaoFundAmount))),
	)
	s.Require().NoError(s.network.NextBlock())

	fundMsg := ucdaotypes.NewMsgFund(sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, ucdaoFundAmount)), owner.AccAddr)
	fundRes, err := s.factory.CommitCosmosTx(owner.Priv, commonfactory.CosmosTxArgs{
		Msgs: []sdk.Msg{fundMsg},
	})
	s.Require().NoError(err)
	s.Require().True(fundRes.IsOK(), "MsgFund failed: %s", fundRes.Log)
	s.Require().NoError(s.network.NextBlock())

	forwarderContract, err := contracts.LoadUcdaoForwarderContract()
	s.Require().NoError(err)
	contractAddr, err := s.factory.DeployContract(
		owner.Priv,
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: forwarderContract},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	spendLimit := sdk.NewCoin(utils.BaseDenom, ucdaoFundAmount)
	transferAuthz, err := ucdaotypes.NewTransferOwnershipAuthorization(&spendLimit)
	s.Require().NoError(err)
	expiration := time.Now().Add(time.Hour)
	s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
		ctx,
		contractAddr.Bytes(),
		owner.AccAddr,
		transferAuthz,
		&expiration,
	))

	beforeOwner := daoKeeper.GetBalance(ctx, owner.AccAddr, utils.BaseDenom).Amount
	beforeNewOwner := daoKeeper.GetBalance(ctx, newOwner.AccAddr, utils.BaseDenom).Amount
	beforeTotal := beforeOwner.Add(beforeNewOwner)
	s.T().Logf("before transfer: owner=%s newOwner=%s total=%s", beforeOwner, beforeNewOwner, beforeTotal)

	transferCalldata, err := s.precompile.ABI.Pack(
		ucdao.TransferOwnershipWithAmountMethod,
		owner.Addr,
		newOwner.Addr,
		[]string{utils.BaseDenom},
		[]*big.Int{ucdaoTransferAmount.BigInt()},
	)
	s.Require().NoError(err)

	callArgs := factory.CallArgs{
		ContractABI: forwarderContract.ABI,
		MethodName:  "touchAndForward",
		Args:        []interface{}{ownerEscrowEVM, ucdaoPrecompileAddr, transferCalldata},
	}
	txArgs := evmtypes.EvmTxArgs{
		To:       &contractAddr,
		GasLimit: 1_500_000,
		Amount:   big.NewInt(1),
	}
	logCheck := testutil.LogCheckArgs{}.WithExpPass(true)

	res, ethRes, err := s.factory.CallContractAndCheckLogs(
		owner.Priv,
		txArgs,
		callArgs,
		logCheck,
	)
	s.Require().NoError(err)
	s.Require().True(res.IsOK(), "touchAndForward failed: %s", res.Log)
	s.Require().False(ethRes.Failed(), "touchAndForward VM error: %s", ethRes.VmError)
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	afterOwner := daoKeeper.GetBalance(ctx, owner.AccAddr, utils.BaseDenom).Amount
	afterNewOwner := daoKeeper.GetBalance(ctx, newOwner.AccAddr, utils.BaseDenom).Amount
	afterTotal := afterOwner.Add(afterNewOwner)
	deltaOwner := afterOwner.Sub(beforeOwner)
	deltaTotal := afterTotal.Sub(beforeTotal)
	s.T().Logf("after transfer: owner=%s newOwner=%s total=%s", afterOwner, afterNewOwner, afterTotal)
	s.T().Logf("after transfer: delta owner=%s delta total=%s (gas used %d)", deltaOwner, deltaTotal, res.GasUsed)

	// Bug: owner escrow is restored to B+1 while newOwner keeps X → total grows by ~X.
	touchDust := math.NewInt(1)
	ownerNotDebited := afterOwner.Sub(beforeOwner.Add(touchDust)).Abs().LTE(touchDust)
	totalInflated := deltaTotal.GTE(ucdaoTransferAmount)
	bugReproduced := ownerNotDebited && afterNewOwner.Equal(ucdaoTransferAmount) && totalInflated
	if bugReproduced {
		s.T().Logf(
			"BUG reproduced: owner escrow %s (before %s) and newOwner %s; total grew by %s (minted transfer amount)",
			afterOwner, beforeOwner, afterNewOwner, deltaTotal,
		)
	}
	s.Require().False(
		bugReproduced,
		fmt.Sprintf(
			"bug: escrow double credit — owner should be %s, got %s; newOwner %s; total delta %s",
			beforeOwner.Add(touchDust).Sub(ucdaoTransferAmount), afterOwner, afterNewOwner, deltaTotal,
		),
	)

	s.Require().Equal(ucdaoTransferAmount, afterNewOwner,
		"newOwner escrow should receive the transferred amount")
	s.Require().Equal(beforeOwner.Add(touchDust).Sub(ucdaoTransferAmount), afterOwner,
		"owner escrow should decrease by the transfer (plus 1 wei touch)")
	s.Require().Equal(beforeTotal.Add(touchDust), afterTotal,
		"total escrow aISLM must stay conserved aside from the 1 wei touch")
}
