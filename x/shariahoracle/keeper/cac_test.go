package keeper_test

import (
	"fmt"

	utiltx "github.com/haqq-network/haqq/testutil/tx"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	"github.com/haqq-network/haqq/x/shariahoracle/keeper"
	shariahoracletypes "github.com/haqq-network/haqq/x/shariahoracle/types"
	"github.com/stretchr/testify/mock"
)

func (suite *KeeperTestSuite) TestDoesAddressHaveCAC() {
	var mockEVMKeeper *MockEVMKeeper
	testCases := []struct {
		name     string
		malleate func()
		expRes   bool
		res      bool
	}{
		{
			"Failed to call Evm",
			func() {
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
			},
			false,
			false,
		},
		{
			"Incorrect res",
			func() {
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: []uint8{0, 0}}, nil).Once()
			},
			false,
			false,
		},
		{
			"Correct Execution",
			func() {
				balance := make([]uint8, 32)
				balance[31] = uint8(1)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
			},
			true,
			true,
		},
	}
	for _, tc := range testCases {
		suite.SetupTest() // reset
		mockEVMKeeper = &MockEVMKeeper{}
		suite.app.ShariaOracleKeeper = keeper.NewKeeper(
			suite.app.GetKey("shariahoracle"),
			suite.app.AppCodec(),
			suite.app.GetSubspace(shariahoracletypes.ModuleName),
			mockEVMKeeper,
			suite.app.AccountKeeper,
		)

		tc.malleate()

		doesHave, err := suite.app.ShariaOracleKeeper.DoesAddressHaveCAC(suite.ctx, utiltx.GenerateAddress().String())
		if tc.res {
			suite.Require().Equal(tc.expRes, doesHave)
		} else {
			suite.Require().Error(err)
		}
	}
}
