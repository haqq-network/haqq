package keeper_test

import (
	"bytes"
	"math/big"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/x/evm/statedb"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// splitInitCode returns CREATE initcode that CALLs `blocked` with 1 wei and
// `credit` with msg.value-1, then returns empty runtime. EVM execution
// succeeds; the blocked-address failure happens later in StateDB.Commit.
func splitInitCode(credit, blocked common.Address) []byte {
	var code []byte
	appendCall := func(addr common.Address, valueOps []byte) {
		code = append(code, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00)
		code = append(code, valueOps...)
		code = append(code, 0x73)
		code = append(code, addr.Bytes()...)
		// Stipend 30k so both CALLs run; GAS would forward almost all remaining
		// gas to the first CALL and starve the second.
		code = append(code, 0x61, 0x75, 0x30, 0xf1, 0x50) // PUSH2 30000, CALL, POP
	}
	appendCall(blocked, []byte{0x60, 0x01})            // 1 wei
	appendCall(credit, []byte{0x34, 0x60, 0x01, 0x03}) // CALLVALUE - 1
	code = append(code, 0x60, 0x00, 0x60, 0x00, 0xf3)  // RETURN empty
	return code
}

func (suite *KeeperTestSuite) TestCommitAtomicityMoneyPrint() {
	suite.SetupTest()

	credit := common.BigToAddress(big.NewInt(10))
	blocked := common.BytesToAddress(authtypes.NewModuleAddress(distrtypes.ModuleName).Bytes())
	debit := common.HexToAddress("0xFFfFfFffFFfffFFfFFfFFFFFffFFFffffFfFFFfF")

	suite.Require().True(bytes.Compare(credit.Bytes(), blocked.Bytes()) < 0, "credit must sort before blocked")
	suite.Require().True(bytes.Compare(blocked.Bytes(), debit.Bytes()) < 0, "blocked must sort before debit")

	k := suite.network.App.EvmKeeper
	bank := suite.network.App.BankKeeper
	denom := suite.network.GetDenom()

	fundAmt := sdkmath.NewInt(1_000_000)
	suite.Require().NoError(suite.network.FundAccountWithBaseDenom(sdk.AccAddress(debit.Bytes()), fundAmt))

	// Take the context after funding: FundAccountWithBaseDenom commits a block, so a
	// context read before it points at the previous state.
	ctx := suite.network.GetContext()

	mintToCredit := big.NewInt(1000)
	mintToBlocked := big.NewInt(1)
	burnFromDebit := big.NewInt(1001)

	suite.Run("direct Commit discards credit when blocked fails", func() {
		creditBefore := bank.GetBalance(ctx, sdk.AccAddress(credit.Bytes()), denom)
		debitBefore := bank.GetBalance(ctx, sdk.AccAddress(debit.Bytes()), denom)

		db := statedb.New(ctx, k, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash())))
		db.AddBalance(credit, mintToCredit)
		db.AddBalance(blocked, mintToBlocked)
		db.SubBalance(debit, burnFromDebit)

		err := db.Commit()
		suite.Require().Error(err, "Commit must fail on blocked module account")
		suite.Require().Contains(err.Error(), "failed to set account")

		creditAfter := bank.GetBalance(ctx, sdk.AccAddress(credit.Bytes()), denom)
		debitAfter := bank.GetBalance(ctx, sdk.AccAddress(debit.Bytes()), denom)

		suite.Require().True(creditAfter.Amount.Equal(creditBefore.Amount),
			"atomic Commit must discard credit: before=%s after=%s err=%v",
			creditBefore, creditAfter, err)
		suite.Require().True(debitAfter.Amount.Equal(debitBefore.Amount),
			"debit must not be applied after the mid-loop failure: before=%s after=%s",
			debitBefore, debitAfter)
	})
}

func (suite *KeeperTestSuite) TestCommitAtomicityCacheContextDiscards() {
	suite.SetupTest()

	credit := common.BigToAddress(big.NewInt(10))
	blocked := common.BytesToAddress(authtypes.NewModuleAddress(distrtypes.ModuleName).Bytes())
	debit := common.HexToAddress("0xFFfFfFffFFfffFFfFFfFFFFFffFFFffffFfFFFfF")

	k := suite.network.App.EvmKeeper
	bank := suite.network.App.BankKeeper
	denom := suite.network.GetDenom()

	suite.Require().NoError(suite.network.FundAccountWithBaseDenom(sdk.AccAddress(debit.Bytes()), sdkmath.NewInt(1_000_000)))

	// Take the context after funding: FundAccountWithBaseDenom commits a block, so a
	// context read before it points at the previous state.
	parent := suite.network.GetContext()

	creditBefore := bank.GetBalance(parent, sdk.AccAddress(credit.Bytes()), denom)
	debitBefore := bank.GetBalance(parent, sdk.AccAddress(debit.Bytes()), denom)

	// Same isolation ApplyTransaction uses: CacheContext is only written on success.
	tmpCtx, write := parent.CacheContext()
	db := statedb.New(tmpCtx, k, statedb.NewEmptyTxConfig(common.BytesToHash(parent.HeaderHash())))
	db.AddBalance(credit, big.NewInt(1000))
	db.AddBalance(blocked, big.NewInt(1))
	db.SubBalance(debit, big.NewInt(1001))
	err := db.Commit()
	suite.Require().Error(err)
	// Intentionally do not call write() — this is what ApplyTransaction does on error.

	creditAfter := bank.GetBalance(parent, sdk.AccAddress(credit.Bytes()), denom)
	debitAfter := bank.GetBalance(parent, sdk.AccAddress(debit.Bytes()), denom)
	_ = write

	suite.Require().True(creditAfter.Amount.Equal(creditBefore.Amount),
		"parent ctx must not see credit after discarded CacheContext: before=%s after=%s",
		creditBefore, creditAfter)
	suite.Require().True(debitAfter.Amount.Equal(debitBefore.Amount),
		"parent ctx must not see debit after discarded CacheContext")
}

func (suite *KeeperTestSuite) TestApplyTransactionDoesNotPrintMoney() {
	suite.SetupTest()

	credit := common.BigToAddress(big.NewInt(10))
	blocked := common.BytesToAddress(authtypes.NewModuleAddress(distrtypes.ModuleName).Bytes())

	var priv *ethsecp256k1.PrivKey
	var from common.Address
	for i := 0; i < 10_000; i++ {
		k, err := ethsecp256k1.GenerateKey()
		suite.Require().NoError(err)
		addr := common.BytesToAddress(k.PubKey().Address().Bytes())
		if bytes.Compare(addr.Bytes(), blocked.Bytes()) > 0 {
			priv, from = k, addr
			break
		}
	}
	suite.Require().NotNil(priv, "failed to grind a sender address after blocked")
	suite.Require().True(bytes.Compare(credit.Bytes(), blocked.Bytes()) < 0)
	suite.Require().True(bytes.Compare(blocked.Bytes(), from.Bytes()) < 0)

	fund := sdkmath.NewInt(10_000_000_000_000_000)
	suite.Require().NoError(suite.network.FundAccountWithBaseDenom(sdk.AccAddress(from.Bytes()), fund))

	bank := suite.network.App.BankKeeper
	denom := suite.network.GetDenom()
	ctx := suite.network.GetContext()
	creditBefore := bank.GetBalance(ctx, sdk.AccAddress(credit.Bytes()), denom)
	fromBefore := bank.GetBalance(ctx, sdk.AccAddress(from.Bytes()), denom)

	value := big.NewInt(1_000_000)
	txArgs := evmtypes.EvmTxArgs{
		Input:    splitInitCode(credit, blocked),
		Amount:   value,
		GasLimit: 1_000_000,
		GasPrice: big.NewInt(1),
	}
	ethTx, err := suite.factory.GenerateSignedMsgEthereumTx(priv, txArgs)
	suite.Require().NoError(err)

	res, err := suite.network.App.EvmKeeper.ApplyTransaction(ctx, ethTx.AsTransaction())
	suite.Require().Error(err, "ApplyTransaction must fail when Commit hits a blocked account")
	suite.T().Logf("ApplyTransaction err=%v", err)
	if res != nil {
		suite.T().Logf("ApplyTransaction res hash=%s vmerr=%s gas=%d", res.Hash, res.VmError, res.GasUsed)
	}

	// ApplyTransaction error path consumes the shared gas meter; restore it for reads.
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	creditAfter := bank.GetBalance(ctx, sdk.AccAddress(credit.Bytes()), denom)
	fromAfter := bank.GetBalance(ctx, sdk.AccAddress(from.Bytes()), denom)

	suite.Require().True(creditAfter.Amount.Equal(creditBefore.Amount),
		"ApplyTransaction CacheContext must discard printed credit: before=%s after=%s err=%v",
		creditBefore, creditAfter, err)
	suite.Require().True(fromAfter.Amount.Equal(fromBefore.Amount),
		"sender must not be debited when ApplyTransaction returns Commit error: before=%s after=%s",
		fromBefore, fromAfter)
}
