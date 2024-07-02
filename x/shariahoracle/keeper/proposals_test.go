package keeper_test

import (
	"fmt"

	"github.com/haqq-network/haqq/contracts"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/x/shariahoracle/keeper"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
	"github.com/stretchr/testify/mock"
)

func (suite *KeeperTestSuite) TestGrantCAC() { //nolint:govet // we can copy locks here because it is a test

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"ok",
			func() {
				cacAddr, err := suite.DeployUUPSContract(contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				params := types.NewParams(cacAddr.String())
				suite.app.ShariahOracleKeeper.SetParams(suite.ctx, params) //nolint:errcheck
			},
			true,
		},
		{
			"fail - CAC already issued",
			func() {
				cacAddr, err := suite.DeployUUPSContract(contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				params := types.NewParams(cacAddr.String())
				suite.app.ShariahOracleKeeper.SetParams(suite.ctx, params) //nolint:errcheck
				err = suite.app.ShariahOracleKeeper.GrantCAC(suite.ctx, types.ModuleAddress.String())
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"fail - force fail evm",
			func() {
				mockERC20Keeper := &MockERC20Keeper{}

				suite.app.ShariahOracleKeeper = keeper.NewKeeper(
					suite.app.GetKey("shariahoracle"), suite.app.AppCodec(),
					suite.app.GetSubspace(types.ModuleName),
					mockERC20Keeper,
					suite.app.AccountKeeper,
				)

				mockERC20Keeper.On("CallEVM",
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything).Return(nil, fmt.Errorf("something went wrong"))
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			tc.malleate()

			err := suite.app.ShariahOracleKeeper.GrantCAC(suite.ctx, types.ModuleAddress.String())
			suite.Commit()

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				minted, err := suite.app.ShariahOracleKeeper.DoesAddressHaveCAC(suite.ctx, types.ModuleAddress.String())
				suite.Require().NoError(err)
				suite.Require().True(minted)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRevokeCAC() { //nolint:govet // we can copy locks here because it is a test

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"ok",
			func() {
				cacAddr, err := suite.DeployUUPSContract(contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				params := types.NewParams(cacAddr.String())
				suite.app.ShariahOracleKeeper.SetParams(suite.ctx, params) //nolint:errcheck
				err = suite.app.ShariahOracleKeeper.GrantCAC(suite.ctx, types.ModuleAddress.String())
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"fail - CAC not issued",
			func() {
				cacAddr, err := suite.DeployUUPSContract(contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				params := types.NewParams(cacAddr.String())
				suite.app.ShariahOracleKeeper.SetParams(suite.ctx, params) //nolint:errcheck
			},
			false,
		},
		{
			"fail - force fail evm",
			func() {
				mockERC20Keeper := &MockERC20Keeper{}

				suite.app.ShariahOracleKeeper = keeper.NewKeeper(
					suite.app.GetKey("shariahoracle"), suite.app.AppCodec(),
					suite.app.GetSubspace(types.ModuleName),
					mockERC20Keeper,
					suite.app.AccountKeeper,
				)

				mockERC20Keeper.On("CallEVM",
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything).Return(nil, fmt.Errorf("something went wrong"))
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			tc.malleate()

			err := suite.app.ShariahOracleKeeper.RevokeCAC(suite.ctx, types.ModuleAddress.String())
			suite.Commit()

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				minted, err := suite.app.ShariahOracleKeeper.DoesAddressHaveCAC(suite.ctx, types.ModuleAddress.String())
				suite.Require().NoError(err)
				suite.Require().False(minted)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateCACContract() { //nolint:govet // we can copy locks here because it is a test

	testCases := []struct {
		name     string
		malleate func() string
		expPass  bool
	}{
		{
			"ok - successful update",
			func() string {
				cacAddr, err := suite.DeployUUPSContract(contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				params := types.NewParams(cacAddr.String())
				suite.app.ShariahOracleKeeper.SetParams(suite.ctx, params) //nolint:errcheck
				newImplementationAddress, err := suite.PrepareProxyUpgrade(contracts.CommunityApprovalCertificatesContract)
				suite.Require().NoError(err)
				return newImplementationAddress.String()
			},
			true,
		},
		{
			"fail - new implementation not deployed",
			func() string {
				cacAddr, err := suite.DeployUUPSContract(contracts.CommunityApprovalCertificatesContract,
					CACModuleAddress,
					CACModuleAddress,
					CACModuleAddress,
					CACContractBaseURI,
				)
				suite.Require().NoError(err)
				params := types.NewParams(cacAddr.String())
				suite.app.ShariahOracleKeeper.SetParams(suite.ctx, params) //nolint:errcheck
				return utiltx.GenerateAddress().String()
			},
			false,
		},
		{
			"fail - force fail evm",
			func() string {
				mockERC20Keeper := &MockERC20Keeper{}

				suite.app.ShariahOracleKeeper = keeper.NewKeeper(
					suite.app.GetKey("shariahoracle"), suite.app.AppCodec(),
					suite.app.GetSubspace(types.ModuleName),
					mockERC20Keeper,
					suite.app.AccountKeeper,
				)

				mockERC20Keeper.On("CallEVM",
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything).Return(nil, fmt.Errorf("something went wrong"))
				return ""
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			newImplementationAddress := tc.malleate()

			err := suite.app.ShariahOracleKeeper.UpdateCACContract(suite.ctx, newImplementationAddress)
			suite.Commit()

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}