package evm_test

import (
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmante "github.com/haqq-network/haqq/app/ante/evm"
	"github.com/haqq-network/haqq/testutil"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var execTypes = []struct {
	name      string
	isCheckTx bool
	simulate  bool
}{
	{"deliverTx", false, false},
	{"deliverTxSimulate", false, true},
}

func (suite *AnteTestSuite) TestEthMinGasPriceDecorator() {
	var ctx sdk.Context
	denom := evmtypes.DefaultEVMDenom
	_, privKey := testutiltx.NewAddrKey()
	to := testutiltx.GenerateAddress()
	emptyAccessList := ethtypes.AccessList{}

	testCases := []struct {
		name     string
		malleate func() sdk.Tx
		expPass  bool
		errMsg   string
	}{
		{
			"invalid tx type",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				return &testutiltx.InvalidTx{}
			},
			false,
			"invalid message type",
		},
		{
			"wrong tx type",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)
				testMsg := banktypes.MsgSend{
					FromAddress: "evmos1x8fhpj9nmhqk8z9kpgjt95ck2xwyue0ptzkucp",
					ToAddress:   "evmos1dx67l23hz9l0k9hcher8xz04uj7wf3yu26l2yn",
					Amount:      sdk.Coins{sdk.Coin{Amount: sdkmath.NewInt(10), Denom: denom}},
				}
				txBuilder := suite.CreateTestCosmosTxBuilder(sdkmath.NewInt(0), denom, &testMsg)
				return txBuilder.GetTx()
			},
			false,
			"invalid message type",
		},
		{
			"valid: invalid tx type with MinGasPrices = 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)
				return &testutiltx.InvalidTx{}
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 0, gasPrice = 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:       &to,
					Nonce:    0,
					Amount:   nil,
					GasLimit: 100000,
					GasPrice: big.NewInt(0),
				})
				suite.Require().NoError(err)

				return tx
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 0, gasPrice > 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:       &to,
					Nonce:    0,
					Amount:   nil,
					GasLimit: 100000,
					GasPrice: big.NewInt(10),
				})
				suite.Require().NoError(err)

				return tx
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 10, gasPrice = 10",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:       &to,
					Nonce:    0,
					Amount:   nil,
					GasLimit: 100000,
					GasPrice: big.NewInt(10),
				})
				suite.Require().NoError(err)

				return tx
			},
			true,
			"",
		},
		{
			"invalid legacy tx with MinGasPrices = 10, gasPrice = 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:       &to,
					Nonce:    0,
					Amount:   nil,
					GasLimit: 100000,
					GasPrice: big.NewInt(0),
				})
				suite.Require().NoError(err)

				return tx
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"valid dynamic tx with MinGasPrices = 0, EffectivePrice = 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:        &to,
					Nonce:     0,
					Amount:    nil,
					GasLimit:  100000,
					GasFeeCap: big.NewInt(0),
					GasTipCap: big.NewInt(0),
					Accesses:  &emptyAccessList,
				})
				suite.Require().NoError(err)

				return tx
			},
			true,
			"",
		},
		{
			"valid dynamic tx with MinGasPrices = 0, EffectivePrice > 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:        &to,
					Nonce:     0,
					Amount:    nil,
					GasLimit:  100000,
					GasFeeCap: big.NewInt(100),
					GasTipCap: big.NewInt(50),
					Accesses:  &emptyAccessList,
				})
				suite.Require().NoError(err)

				return tx
			},
			true,
			"",
		},
		{
			"valid dynamic tx with MinGasPrices < EffectivePrice",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:        &to,
					Nonce:     0,
					Amount:    nil,
					GasLimit:  100000,
					GasFeeCap: big.NewInt(100),
					GasTipCap: big.NewInt(100),
					Accesses:  &emptyAccessList,
				})
				suite.Require().NoError(err)

				return tx
			},
			true,
			"",
		},
		{
			"invalid dynamic tx with MinGasPrices > EffectivePrice",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:        &to,
					Nonce:     0,
					Amount:    nil,
					GasLimit:  100000,
					GasFeeCap: big.NewInt(0),
					GasTipCap: big.NewInt(0),
					Accesses:  &emptyAccessList,
				})
				suite.Require().NoError(err)

				return tx
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"invalid dynamic tx with MinGasPrices > BaseFee, MinGasPrices > EffectivePrice",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(100)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				feemarketParams := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				feemarketParams.BaseFee = sdkmath.NewInt(10)
				err = suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, feemarketParams)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:        &to,
					Nonce:     0,
					Amount:    nil,
					GasLimit:  100000,
					GasFeeCap: big.NewInt(1000),
					GasTipCap: big.NewInt(0),
					Accesses:  &emptyAccessList,
				})
				suite.Require().NoError(err)

				return tx
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"valid dynamic tx with MinGasPrices > BaseFee, MinGasPrices < EffectivePrice (big GasTipCap)",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(100)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				feemarketParams := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				feemarketParams.BaseFee = sdkmath.NewInt(10)
				err = suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, feemarketParams)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:        &to,
					Nonce:     0,
					Amount:    nil,
					GasLimit:  100000,
					GasFeeCap: big.NewInt(1000),
					GasTipCap: big.NewInt(101),
					Accesses:  &emptyAccessList,
				})
				suite.Require().NoError(err)

				return tx
			},
			true,
			"",
		},
		{
			"panic bug, requiredFee > math.MaxInt64",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(ctx)
				params.MinGasPrice = sdkmath.LegacyNewDec(math.MaxInt64)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)

				tx, err := suite.GetTxFactory().GenerateSignedEthTx(privKey, evmtypes.EvmTxArgs{
					ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					To:        &to,
					Nonce:     0,
					Amount:    nil,
					GasLimit:  100000,
					GasFeeCap: big.NewInt(math.MaxInt64),
					GasTipCap: big.NewInt(100),
					Accesses:  &emptyAccessList,
				})
				suite.Require().NoError(err)

				return tx
			},
			false,
			"provided fee < minimum global fee",
		},
	}

	for _, et := range execTypes {
		for _, tc := range testCases {
			suite.Run(et.name+"_"+tc.name, func() {
				suite.SetupTest()
				ctx = suite.GetNetwork().GetContext()
				dec := evmante.NewEthMinGasPriceDecorator(suite.GetNetwork().App.FeeMarketKeeper, suite.GetNetwork().App.EvmKeeper)
				_, err := dec.AnteHandle(ctx, tc.malleate(), et.simulate, testutil.NextFn)

				if tc.expPass {
					suite.Require().NoError(err, tc.name)
				} else {
					suite.Require().Error(err, tc.name)
					suite.Require().Contains(err.Error(), tc.errMsg, tc.name)
				}
			})
		}
	}
}

func (suite *AnteTestSuite) TestEthMempoolFeeDecorator() {
	// TODO: add test
}
