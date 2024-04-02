package evm_test

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmante "github.com/haqq-network/haqq/app/ante/evm"
	antetestutils "github.com/haqq-network/haqq/app/ante/testutils"
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
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				return &testutiltx.InvalidTx{}
			},
			false,
			"invalid transaction",
		},
		{
			"wrong tx type",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)
				testMsg := banktypes.MsgSend{
					FromAddress: "haqq1tjdjfavsy956d25hvhs3p0nw9a7pfghqm0up92",
					ToAddress:   "haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
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
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)
				return &testutiltx.InvalidTx{}
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 0, gasPrice = 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:    0,
					To:       &to,
					GasLimit: antetestutils.TestGasLimit,
					GasPrice: big.NewInt(0),
				}
				//  from common.Address,
				//	to common.Address,
				//	amount *big.Int,
				//	input []byte,
				//	gasPrice *big.Int,
				//	gasFeeCap *big.Int,
				//	gasTipCap *big.Int,
				//	accesses *ethtypes.AccessList,
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), big.NewInt(0), nil, nil, nil)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 0, gasPrice > 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:    0,
					To:       &to,
					GasLimit: antetestutils.TestGasLimit,
					GasPrice: big.NewInt(10),
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), big.NewInt(10), nil, nil, nil)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 10, gasPrice = 10",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:    0,
					To:       &to,
					GasLimit: antetestutils.TestGasLimit,
					GasPrice: big.NewInt(10),
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), big.NewInt(10), nil, nil, nil)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			true,
			"",
		},
		{
			"invalid legacy tx with MinGasPrices = 10, gasPrice = 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:    0,
					To:       &to,
					GasLimit: antetestutils.TestGasLimit,
					GasPrice: big.NewInt(0),
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), big.NewInt(0), nil, nil, nil)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"valid dynamic tx with MinGasPrices = 0, EffectivePrice = 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:     0,
					To:        &to,
					GasLimit:  antetestutils.TestGasLimit,
					GasFeeCap: big.NewInt(0),
					GasTipCap: big.NewInt(0),
					Accesses:  &emptyAccessList,
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(0), big.NewInt(0), &emptyAccessList)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			true,
			"",
		},
		{
			"valid dynamic tx with MinGasPrices = 0, EffectivePrice > 0",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyZeroDec()
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:     0,
					To:        &to,
					GasLimit:  antetestutils.TestGasLimit,
					GasFeeCap: big.NewInt(100),
					GasTipCap: big.NewInt(50),
					Accesses:  &emptyAccessList,
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(100), big.NewInt(50), &emptyAccessList)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			true,
			"",
		},
		{
			"valid dynamic tx with MinGasPrices < EffectivePrice",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:     0,
					To:        &to,
					GasLimit:  antetestutils.TestGasLimit,
					GasFeeCap: big.NewInt(100),
					GasTipCap: big.NewInt(100),
					Accesses:  &emptyAccessList,
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(100), big.NewInt(100), &emptyAccessList)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			true,
			"",
		},
		{
			"invalid dynamic tx with MinGasPrices > EffectivePrice",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyNewDec(10)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:     0,
					To:        &to,
					GasLimit:  antetestutils.TestGasLimit,
					GasFeeCap: big.NewInt(0),
					GasTipCap: big.NewInt(0),
					Accesses:  &emptyAccessList,
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(0), big.NewInt(0), &emptyAccessList)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"invalid dynamic tx with MinGasPrices > BaseFee, MinGasPrices > EffectivePrice",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyNewDec(100)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				feemarketParams := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				feemarketParams.BaseFee = sdkmath.NewInt(10)
				err = suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), feemarketParams)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:     0,
					To:        &to,
					GasLimit:  antetestutils.TestGasLimit,
					GasFeeCap: big.NewInt(1000),
					GasTipCap: big.NewInt(0),
					Accesses:  &emptyAccessList,
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(1000), big.NewInt(0), &emptyAccessList)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"valid dynamic tx with MinGasPrices > BaseFee, MinGasPrices < EffectivePrice (big GasTipCap)",
			func() sdk.Tx {
				params := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				params.MinGasPrice = sdkmath.LegacyNewDec(100)
				err := suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), params)
				suite.Require().NoError(err)

				feemarketParams := suite.GetNetwork().App.FeeMarketKeeper.GetParams(suite.GetNetwork().GetContext())
				feemarketParams.BaseFee = sdkmath.NewInt(10)
				err = suite.GetNetwork().App.FeeMarketKeeper.SetParams(suite.GetNetwork().GetContext(), feemarketParams)
				suite.Require().NoError(err)

				txArgs := evmtypes.EvmTxArgs{
					//ChainID:   suite.GetNetwork().App.EvmKeeper.ChainID(),
					//Nonce:     0,
					To:        &to,
					GasLimit:  antetestutils.TestGasLimit,
					GasFeeCap: big.NewInt(1000),
					GasTipCap: big.NewInt(101),
					Accesses:  &emptyAccessList,
				}
				//msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(1000), big.NewInt(101), &emptyAccessList)
				return suite.CreateTxBuilder(privKey, txArgs).GetTx()
			},
			true,
			"",
		},
	}

	for _, et := range execTypes {
		for _, tc := range testCases {
			suite.Run(et.name+"_"+tc.name, func() {
				// s.SetupTest(et.isCheckTx)
				suite.SetupTest()
				dec := evmante.NewEthMinGasPriceDecorator(suite.GetNetwork().App.FeeMarketKeeper, suite.GetNetwork().App.EvmKeeper)
				_, err := dec.AnteHandle(suite.GetNetwork().GetContext(), tc.malleate(), et.simulate, testutil.NextFn)

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
