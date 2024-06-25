package keeper_test

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/contracts"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
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

		erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
		contract, err := suite.DeployUUPSContract(
			contracts.CommunityApprovalCertificatesContract,
			CACModuleAddress,
			CACModuleAddress,
			CACModuleAddress,
			CACContractBaseURI,
		)
		suite.Require().NoError(err)
		account := utiltx.GenerateAddress()

		res, err := suite.app.ShariaOracleKeeper.CallEVM(suite.ctx, erc20, types.ModuleAddress, contract, true, tc.method, account)
		if tc.expPass {
			suite.Require().IsTypef(&evmtypes.MsgEthereumTxResponse{}, res, tc.name)
			suite.Require().NoError(err)
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *KeeperTestSuite) TestCallEVMWithData() {
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	testCases := []struct {
		name     string
		from     common.Address
		malleate func() ([]byte, *common.Address)
		expPass  bool
	}{
		{
			"unknown method",
			types.ModuleAddress,
			func() ([]byte, *common.Address) {
				contract, err := suite.DeployUUPSContract(
					contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				account := utiltx.GenerateAddress()
				data, _ := erc20.Pack("", account)
				return data, &contract
			},
			false,
		},
		{
			"pass",
			types.ModuleAddress,
			func() ([]byte, *common.Address) {
				contract, err := suite.DeployUUPSContract(
					contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				account := utiltx.GenerateAddress()
				data, _ := erc20.Pack("balanceOf", account)
				return data, &contract
			},
			true,
		},
		{
			"fail empty data",
			types.ModuleAddress,
			func() ([]byte, *common.Address) {
				contract, err := suite.DeployUUPSContract(
					contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				return []byte{}, &contract
			},
			false,
		},

		{
			"fail empty sender",
			common.Address{},
			func() ([]byte, *common.Address) {
				contract, err := suite.DeployUUPSContract(
					contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				return []byte{}, &contract
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			data, contract := tc.malleate()

			res, err := suite.app.ShariaOracleKeeper.CallEVMWithData(suite.ctx, tc.from, contract, data, true)
			if tc.expPass {
				suite.Require().IsTypef(&evmtypes.MsgEthereumTxResponse{}, res, tc.name)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
