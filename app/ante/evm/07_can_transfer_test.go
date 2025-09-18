package evm_test

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/app/ante/evm"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func (suite *EvmAnteTestSuite) TestCanTransfer() {
	keyring := testkeyring.New(1)
	unitNetwork := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	grpcHandler := grpc.NewIntegrationHandler(unitNetwork)
	txFactory := factory.New(unitNetwork, grpcHandler)
	senderKey := keyring.GetKey(0)

	testCases := []struct {
		name          string
		expectedError error
		isLondon      bool
		malleate      func(txArgs *evmtypes.EvmTxArgs)
	}{
		{
			name:          "fail: isLondon and insufficient fee",
			expectedError: errortypes.ErrInsufficientFee,
			isLondon:      true,
			malleate: func(txArgs *evmtypes.EvmTxArgs) {
				txArgs.GasFeeCap = big.NewInt(0)
			},
		},
		{
			name:          "fail: invalid tx with insufficient balance",
			expectedError: errortypes.ErrInsufficientFunds,
			isLondon:      true,
			malleate: func(txArgs *evmtypes.EvmTxArgs) {
				balanceResp, err := grpcHandler.GetBalance(senderKey.AccAddr, unitNetwork.GetDenom())
				suite.Require().NoError(err)
				invalidAmount := balanceResp.Balance.Amount.Add(math.NewInt(1)).BigInt()
				txArgs.Amount = invalidAmount
			},
		},
		{
			name:          "success: valid tx and sufficient balance",
			expectedError: nil,
			isLondon:      true,
			malleate: func(*evmtypes.EvmTxArgs) {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("%v_%v", evmtypes.GetTxTypeName(suite.ethTxType), tc.name), func() {
			baseFeeResp, err := grpcHandler.GetBaseFee()
			suite.Require().NoError(err)
			ethCfg := unitNetwork.GetEVMChainConfig()
			evmParams, err := grpcHandler.GetEvmParams()
			suite.Require().NoError(err)
			ctx := unitNetwork.GetContext()
			signer := gethtypes.MakeSigner(ethCfg, big.NewInt(ctx.BlockHeight()))
			txArgs, err := txFactory.GenerateDefaultTxTypeArgs(senderKey.Addr, suite.ethTxType)
			suite.Require().NoError(err)
			txArgs.Amount = big.NewInt(100)

			tc.malleate(&txArgs)

			msg := evmtypes.NewTx(&txArgs)
			msg.From = senderKey.Addr.String()
			signMsg, err := txFactory.SignMsgEthereumTx(senderKey.Priv, *msg)
			suite.Require().NoError(err)
			coreMsg, err := signMsg.AsMessage(signer, baseFeeResp.BaseFee.BigInt())
			suite.Require().NoError(err)

			// Function under test
			err = evm.CanTransfer(
				unitNetwork.GetContext(),
				unitNetwork.App.EvmKeeper,
				coreMsg,
				baseFeeResp.BaseFee.BigInt(),
				ethCfg,
				evmParams.Params,
				tc.isLondon,
			)

			if tc.expectedError != nil {
				suite.Require().Error(err)
				suite.Contains(err.Error(), tc.expectedError.Error())

			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
