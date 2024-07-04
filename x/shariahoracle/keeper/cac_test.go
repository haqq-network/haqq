package keeper_test

import (
	"fmt"

	utiltx "github.com/haqq-network/haqq/testutil/tx"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	"github.com/haqq-network/haqq/x/shariahoracle/keeper"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
	"github.com/stretchr/testify/mock"
)

func (suite *KeeperTestSuite) TestDoesAddressHaveCAC() {
	var mockERC20Keeper *MockERC20Keeper
	testCases := []struct {
		name     string
		malleate func()
		expRes   bool
		res      bool
	}{
		{
			"Failed to call Evm",
			func() {
				mockERC20Keeper.On("CallEVM",
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything).Return(nil, fmt.Errorf("something went wrong")).Once()
			},
			false,
			false,
		},
		{
			"Incorrect res",
			func() {
				mockERC20Keeper.On("CallEVM",
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: []uint8{0, 0}}, nil).Once()
			},
			false,
			false,
		},
		{
			"Correct Execution",
			func() {
				balance := make([]uint8, 32)
				balance[31] = uint8(1)
				mockERC20Keeper.On("CallEVM",
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything,
					mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
			},
			true,
			true,
		},
	}
	for _, tc := range testCases {
		suite.SetupTest() // reset
		mockERC20Keeper = &MockERC20Keeper{}
		suite.app.ShariahOracleKeeper = keeper.NewKeeper(
			suite.app.GetKey("shariahoracle"),
			suite.app.AppCodec(),
			suite.app.GetSubspace(types.ModuleName),
			mockERC20Keeper,
			suite.app.AccountKeeper,
		)

		tc.malleate()

		doesHave, err := suite.app.ShariahOracleKeeper.DoesAddressHaveCAC(suite.ctx, utiltx.GenerateAddress().String())
		if tc.res {
			suite.Require().Equal(tc.expRes, doesHave)
		} else {
			suite.Require().Error(err)
		}
	}
}
