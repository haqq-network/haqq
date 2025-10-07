package keeper_test

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/x/erc20/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func (suite *KeeperTestSuite) TestCallEVM() {
	testCases := []struct {
		name    string
		method  string
		expPass bool
	}{
		{
			"unknown method",
			"",
			false,
		},
		{
			"pass",
			"balanceOf",
			true,
		},
	}
	for _, tc := range testCases {
		suite.SetupTest() // reset
		// We don't have native WISLM precompiled contract on chain by default, so deploy custom one
		erc20 := contracts.ERC20MinterBurnerDecimalsContract
		wislmContract, err := suite.factory.DeployContract(
			suite.keyring.GetPrivKey(0),
			evmtypes.EvmTxArgs{
				GasPrice: big.NewInt(1e9),
			}, // NOTE: passing empty struct to use default values
			factory.ContractDeploymentData{
				Contract:        erc20,
				ConstructorArgs: []interface{}{"Wrapped ISLM", "WISLM", uint8(18)},
			},
		)
		suite.Require().NoError(err)

		account := utiltx.GenerateAddress()
		res, err := suite.network.App.EvmKeeper.CallEVM(suite.network.GetContext(), erc20.ABI, types.ModuleAddress, wislmContract, false, tc.method, account)
		if tc.expPass {
			suite.Require().IsTypef(&evmtypes.MsgEthereumTxResponse{}, res, tc.name)
			suite.Require().NoError(err)
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *KeeperTestSuite) TestCallEVMWithData() {
	erc20 := contracts.ERC20MinterBurnerDecimalsContract
	testCases := []struct {
		name     string
		from     common.Address
		malleate func() []byte
		deploy   bool
		expPass  bool
	}{
		{
			"unknown method",
			types.ModuleAddress,
			func() []byte {
				account := utiltx.GenerateAddress()
				data, _ := erc20.ABI.Pack("", account)
				return data
			},
			false,
			false,
		},
		{
			"pass",
			types.ModuleAddress,
			func() []byte {
				account := utiltx.GenerateAddress()
				data, _ := erc20.ABI.Pack("balanceOf", account)
				return data
			},
			false,
			true,
		},
		{
			"fail empty data",
			types.ModuleAddress,
			func() []byte {
				return []byte{}
			},
			false,
			false,
		},

		{
			"fail empty sender",
			common.Address{},
			func() []byte {
				return []byte{}
			},
			false,
			false,
		},
		{
			"deploy",
			types.ModuleAddress,
			func() []byte {
				ctorArgs, _ := erc20.ABI.Pack("", "test", "test", uint8(18))
				data := append(erc20.Bin, ctorArgs...) //nolint:gocritic
				return data
			},
			true,
			true,
		},
		{
			"fail deploy",
			types.ModuleAddress,
			func() []byte {
				params := suite.network.App.EvmKeeper.GetParams(suite.network.GetContext())
				params.AccessControl.Create = evmtypes.AccessControlType{
					AccessType: evmtypes.AccessTypeRestricted,
				}
				_ = suite.network.App.EvmKeeper.SetParams(suite.network.GetContext(), params)
				ctorArgs, _ := erc20.ABI.Pack("", "test", "test", uint8(18))
				data := append(erc20.Bin, ctorArgs...) //nolint:gocritic
				return data
			},
			true,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			// We don't have native WISLM precompiled contract on chain by default, so deploy custom one
			wislmContract, err := suite.factory.DeployContract(
				suite.keyring.GetPrivKey(0),
				evmtypes.EvmTxArgs{
					GasPrice: big.NewInt(1e9),
				}, // NOTE: passing empty struct to use default values
				factory.ContractDeploymentData{
					Contract:        erc20,
					ConstructorArgs: []interface{}{"Wrapped ISLM", "WISLM", uint8(18)},
				},
			)
			suite.Require().NoError(err)

			data := tc.malleate()
			var res *evmtypes.MsgEthereumTxResponse

			if tc.deploy {
				res, err = suite.network.App.EvmKeeper.CallEVMWithData(suite.network.GetContext(), tc.from, nil, data, true)
			} else {
				res, err = suite.network.App.EvmKeeper.CallEVMWithData(suite.network.GetContext(), tc.from, &wislmContract, data, false)
			}

			if tc.expPass {
				suite.Require().IsTypef(&evmtypes.MsgEthereumTxResponse{}, res, tc.name)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
