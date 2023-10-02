package types_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
)

type TxDataTestSuite struct {
	suite.Suite

	sdkInt         sdkmath.Int
	uint64         uint64
	hexUint64      hexutil.Uint64
	bigInt         *big.Int
	hexBigInt      hexutil.Big
	overflowBigInt *big.Int
	sdkZeroInt     sdkmath.Int
	sdkMinusOneInt sdkmath.Int
	invalidAddr    string
	addr           common.Address
	hexAddr        string
	hexDataBytes   hexutil.Bytes
	hexInputBytes  hexutil.Bytes
}

func (suite *TxDataTestSuite) SetupTest() {
	suite.sdkInt = sdkmath.NewInt(9001)
	suite.uint64 = suite.sdkInt.Uint64()
	suite.hexUint64 = hexutil.Uint64(100)
	suite.bigInt = big.NewInt(1)
	suite.hexBigInt = hexutil.Big(*big.NewInt(1))
	suite.overflowBigInt = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(256), nil)
	suite.sdkZeroInt = sdk.ZeroInt()
	suite.sdkMinusOneInt = sdkmath.NewInt(-1)
	suite.invalidAddr = "123456"
	suite.addr = utiltx.GenerateAddress()
	suite.hexAddr = suite.addr.Hex()
	suite.hexDataBytes = hexutil.Bytes([]byte("data"))
	suite.hexInputBytes = hexutil.Bytes([]byte("input"))
}

func TestTxDataTestSuite(t *testing.T) {
	suite.Run(t, new(TxDataTestSuite))
}

func (suite *TxDataTestSuite) TestNewDynamicFeeTx() {
	testCases := []struct {
		name     string
		expError bool
		tx       *ethtypes.Transaction
	}{
		{
			"non-empty tx",
			false,
			ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				Value:      big.NewInt(1),
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
		},
		{
			"value out of bounds tx",
			true,
			ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				Value:      suite.overflowBigInt,
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
		},
		{
			"gas fee cap out of bounds tx",
			true,
			ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				GasFeeCap:  suite.overflowBigInt,
				Value:      big.NewInt(1),
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
		},
		{
			"gas tip cap out of bounds tx",
			true,
			ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				GasTipCap:  suite.overflowBigInt,
				Value:      big.NewInt(1),
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
		},
	}
	for _, tc := range testCases {
		tx, err := evmtypes.NewDynamicFeeTx(tc.tx)

		if tc.expError {
			suite.Require().Error(err)
		} else {
			suite.Require().NoError(err)
			suite.Require().NotEmpty(tx)
			suite.Require().Equal(uint8(2), tx.TxType())
		}
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxAsEthereumData() {
	feeConfig := &ethtypes.DynamicFeeTx{
		Nonce:      1,
		Data:       []byte("data"),
		Gas:        100,
		Value:      big.NewInt(1),
		AccessList: ethtypes.AccessList{},
		To:         &suite.addr,
		V:          suite.bigInt,
		R:          suite.bigInt,
		S:          suite.bigInt,
	}

	tx := ethtypes.NewTx(feeConfig)

	dynamicFeeTx, err := evmtypes.NewDynamicFeeTx(tx)
	suite.Require().NoError(err)

	res := dynamicFeeTx.AsEthereumData()
	resTx := ethtypes.NewTx(res)

	suite.Require().Equal(feeConfig.Nonce, resTx.Nonce())
	suite.Require().Equal(feeConfig.Data, resTx.Data())
	suite.Require().Equal(feeConfig.Gas, resTx.Gas())
	suite.Require().Equal(feeConfig.Value, resTx.Value())
	suite.Require().Equal(feeConfig.AccessList, resTx.AccessList())
	suite.Require().Equal(feeConfig.To, resTx.To())
}

func (suite *TxDataTestSuite) TestDynamicFeeTxCopy() {
	tx := &evmtypes.DynamicFeeTx{}
	txCopy := tx.Copy()

	suite.Require().Equal(&evmtypes.DynamicFeeTx{}, txCopy)
	// TODO: Test for different pointers
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetChainID() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"empty chainID",
			evmtypes.DynamicFeeTx{
				ChainID: nil,
			},
			nil,
		},
		{
			"non-empty chainID",
			evmtypes.DynamicFeeTx{
				ChainID: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetChainID()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetAccessList() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  ethtypes.AccessList
	}{
		{
			"empty accesses",
			evmtypes.DynamicFeeTx{
				Accesses: nil,
			},
			nil,
		},
		{
			"nil",
			evmtypes.DynamicFeeTx{
				Accesses: evmtypes.NewAccessList(nil),
			},
			nil,
		},
		{
			"non-empty accesses",
			evmtypes.DynamicFeeTx{
				Accesses: evmtypes.AccessList{
					{
						Address:     suite.hexAddr,
						StorageKeys: []string{},
					},
				},
			},
			ethtypes.AccessList{
				{
					Address:     suite.addr,
					StorageKeys: []common.Hash{},
				},
			},
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetAccessList()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetData() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
	}{
		{
			"non-empty transaction",
			evmtypes.DynamicFeeTx{
				Data: nil,
			},
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetData()

		suite.Require().Equal(tc.tx.Data, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGas() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  uint64
	}{
		{
			"non-empty gas",
			evmtypes.DynamicFeeTx{
				GasLimit: suite.uint64,
			},
			suite.uint64,
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGas()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasPrice() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"non-empty gasFeeCap",
			evmtypes.DynamicFeeTx{
				GasFeeCap: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasPrice()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasTipCap() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"empty gasTipCap",
			evmtypes.DynamicFeeTx{
				GasTipCap: nil,
			},
			nil,
		},
		{
			"non-empty gasTipCap",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasTipCap()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasFeeCap() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"empty gasFeeCap",
			evmtypes.DynamicFeeTx{
				GasFeeCap: nil,
			},
			nil,
		},
		{
			"non-empty gasFeeCap",
			evmtypes.DynamicFeeTx{
				GasFeeCap: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasFeeCap()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetValue() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			"empty amount",
			evmtypes.DynamicFeeTx{
				Amount: nil,
			},
			nil,
		},
		{
			"non-empty amount",
			evmtypes.DynamicFeeTx{
				Amount: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetValue()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetNonce() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  uint64
	}{
		{
			"non-empty nonce",
			evmtypes.DynamicFeeTx{
				Nonce: suite.uint64,
			},
			suite.uint64,
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetNonce()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetTo() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *common.Address
	}{
		{
			"empty suite.address",
			evmtypes.DynamicFeeTx{
				To: "",
			},
			nil,
		},
		{
			"non-empty suite.address",
			evmtypes.DynamicFeeTx{
				To: suite.hexAddr,
			},
			&suite.addr,
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetTo()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxSetSignatureValues() {
	testCases := []struct {
		name    string
		chainID *big.Int
		r       *big.Int
		v       *big.Int
		s       *big.Int
	}{
		{
			"empty values",
			nil,
			nil,
			nil,
			nil,
		},
		{
			"non-empty values",
			suite.bigInt,
			suite.bigInt,
			suite.bigInt,
			suite.bigInt,
		},
	}

	for _, tc := range testCases {
		tx := &evmtypes.DynamicFeeTx{}
		tx.SetSignatureValues(tc.chainID, tc.v, tc.r, tc.s)

		v, r, s := tx.GetRawSignatureValues()
		chainID := tx.GetChainID()

		suite.Require().Equal(tc.v, v, tc.name)
		suite.Require().Equal(tc.r, r, tc.name)
		suite.Require().Equal(tc.s, s, tc.name)
		suite.Require().Equal(tc.chainID, chainID, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxValidate() {
	testCases := []struct {
		name     string
		tx       evmtypes.DynamicFeeTx
		expError bool
	}{
		{
			"empty",
			evmtypes.DynamicFeeTx{},
			true,
		},
		{
			"gas tip cap is nil",
			evmtypes.DynamicFeeTx{
				GasTipCap: nil,
			},
			true,
		},
		{
			"gas fee cap is nil",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkZeroInt,
			},
			true,
		},
		{
			"gas tip cap is negative",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkMinusOneInt,
				GasFeeCap: &suite.sdkZeroInt,
			},
			true,
		},
		{
			"gas tip cap is negative",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkZeroInt,
				GasFeeCap: &suite.sdkMinusOneInt,
			},
			true,
		},
		{
			"gas fee cap < gas tip cap",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkZeroInt,
			},
			true,
		},
		{
			"amount is negative",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkMinusOneInt,
			},
			true,
		},
		{
			"to suite.address is invalid",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.invalidAddr,
			},
			true,
		},
		{
			"chain ID not present on AccessList txs",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.hexAddr,
				ChainID:   nil,
			},
			true,
		},
		{
			"no errors",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.hexAddr,
				ChainID:   &suite.sdkInt,
			},
			false,
		},
	}

	for _, tc := range testCases {
		err := tc.tx.Validate()

		if tc.expError {
			suite.Require().Error(err, tc.name)
			continue
		}

		suite.Require().NoError(err, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveGasPrice() {
	testCases := []struct {
		name    string
		tx      evmtypes.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty dynamic fee tx",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.EffectiveGasPrice(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveFee() {
	testCases := []struct {
		name    string
		tx      evmtypes.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty dynamic fee tx",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				GasLimit:  uint64(1),
			},
			(&suite.sdkInt).BigInt(),
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.EffectiveFee(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveCost() {
	testCases := []struct {
		name    string
		tx      evmtypes.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			"non-empty dynamic fee tx",
			evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				GasLimit:  uint64(1),
				Amount:    &suite.sdkZeroInt,
			},
			(&suite.sdkInt).BigInt(),
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.EffectiveCost(tc.baseFee)

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxFeeCost() {
	tx := &evmtypes.DynamicFeeTx{}
	suite.Require().Panics(func() { tx.Fee() }, "should panic")
	suite.Require().Panics(func() { tx.Cost() }, "should panic")
}
