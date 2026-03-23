package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/utils"

	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	ethiqkeeper "github.com/haqq-network/haqq/x/ethiq/keeper"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

func (suite *KeeperTestSuite) TestMsgMintHaqq() {
	testCases := []struct {
		name        string
		malleate    func(ctx sdk.Context) *ethiqtypes.MsgMintHaqq
		expErr      bool
		errContains string
	}{
		{
			name: "fail - module disabled",
			malleate: func(ctx sdk.Context) *ethiqtypes.MsgMintHaqq {
				// Disable module
				params := s.network.App.EthiqKeeper.GetParams(ctx)
				params.Enabled = false
				s.network.App.EthiqKeeper.SetParams(ctx, params)

				return &ethiqtypes.MsgMintHaqq{
					FromAddress: s.keyring.GetAccAddr(0).String(),
					ToAddress:   s.keyring.GetAccAddr(1).String(),
					IslmAmount:  sdkmath.NewInt(1e18),
				}
			},
			expErr:      true,
			errContains: "module is disabled",
		},
		{
			name: "fail - invalid sender address",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqq {
				return &ethiqtypes.MsgMintHaqq{
					FromAddress: "haqq1gcwaegn",
					ToAddress:   s.keyring.GetAccAddr(1).String(),
					IslmAmount:  sdkmath.NewInt(1e18),
				}
			},
			expErr:      true,
			errContains: "invalid from_address",
		},
		{
			name: "fail - invalid receiver address",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqq {
				return &ethiqtypes.MsgMintHaqq{
					FromAddress: s.keyring.GetAccAddr(0).String(),
					ToAddress:   "haqq1gcwaegn",
					IslmAmount:  sdkmath.NewInt(1e18),
				}
			},
			expErr:      true,
			errContains: "invalid to_address",
		},
		{
			name: "fail - zero amount",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqq {
				return &ethiqtypes.MsgMintHaqq{
					FromAddress: s.keyring.GetAccAddr(0).String(),
					ToAddress:   s.keyring.GetAccAddr(1).String(),
					IslmAmount:  sdkmath.ZeroInt(),
				}
			},
			expErr:      true,
			errContains: "islm_amount must be positive and greater than zero",
		},
		{
			name: "fail - insufficient funds",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqq {
				return &ethiqtypes.MsgMintHaqq{
					FromAddress: s.keyring.GetAccAddr(0).String(),
					ToAddress:   s.keyring.GetAccAddr(1).String(),
					IslmAmount:  sdkmath.NewInt(1e18).MulRaw(1e7),
				}
			},
			expErr:      true,
			errContains: "insufficient funds",
		},
		{
			name: "success - burnt",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqq {
				return &ethiqtypes.MsgMintHaqq{
					FromAddress: s.keyring.GetAccAddr(0).String(),
					ToAddress:   s.keyring.GetAccAddr(1).String(),
					IslmAmount:  sdkmath.NewInt(1e18),
				}
			},
			expErr:      false,
			errContains: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()

			msg := tc.malleate(ctx)
			msgSrv := ethiqkeeper.NewMsgServerImpl(s.network.App.EthiqKeeper)

			_, err := msgSrv.MintHaqq(ctx, msg)
			if tc.expErr {
				suite.Require().Error(err)
				if tc.errContains != "" {
					suite.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgMintHaqqByApplication() {
	testCases := []struct {
		name        string
		malleate    func(ctx sdk.Context) *ethiqtypes.MsgMintHaqqByApplication
		expErr      bool
		errContains string
	}{
		{
			name: "fail - module disabled",
			malleate: func(ctx sdk.Context) *ethiqtypes.MsgMintHaqqByApplication {
				params := s.network.App.EthiqKeeper.GetParams(ctx)
				params.Enabled = false
				s.network.App.EthiqKeeper.SetParams(ctx, params)

				return &ethiqtypes.MsgMintHaqqByApplication{
					FromAddress:   s.keyring.GetAccAddr(0).String(),
					ApplicationId: 0,
				}
			},
			expErr:      true,
			errContains: "module is disabled",
		},
		{
			name: "fail - invalid sender address",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqqByApplication {
				return &ethiqtypes.MsgMintHaqqByApplication{
					FromAddress:   "haqq1gcwaegn",
					ApplicationId: 1,
				}
			},
			expErr:      true,
			errContains: "invalid from_address",
		},
		{
			name: "fail - invalid application ID",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqqByApplication {
				return &ethiqtypes.MsgMintHaqqByApplication{
					FromAddress:   s.keyring.GetAccAddr(0).String(),
					ApplicationId: 99999,
				}
			},
			expErr:      true,
			errContains: "application 99999 not found",
		},
		{
			name: "fail - insufficient funds",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqqByApplication {
				suite.Require().NoError(s.network.FundAccount(
					sdk.MustAccAddressFromBech32("haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z"),
					sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt())),
				))

				return &ethiqtypes.MsgMintHaqqByApplication{
					FromAddress:   "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z",
					ApplicationId: 6,
				}
			},
			expErr:      true,
			errContains: "insufficient funds",
		},
		{
			name: "success - application executed",
			malleate: func(_ sdk.Context) *ethiqtypes.MsgMintHaqqByApplication {
				suite.Require().NoError(s.network.FundAccount(
					sdk.MustAccAddressFromBech32("haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z"),
					sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt().MulRaw(1e18).MulRaw(100))),
				))

				return &ethiqtypes.MsgMintHaqqByApplication{
					FromAddress:   "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z",
					ApplicationId: 6,
				}
			},
			expErr:      false,
			errContains: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()

			msg := tc.malleate(ctx)
			msgSrv := ethiqkeeper.NewMsgServerImpl(s.network.App.EthiqKeeper)

			_, err := msgSrv.MintHaqqByApplication(ctx, msg)
			if tc.expErr {
				suite.Require().Error(err)
				if tc.errContains != "" {
					suite.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgMintHaqqByApplicationAlreadyExecuted() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	// Application 1: fromAddress = "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", burnAmount = 10e18
	// Use application ID 1 to avoid the zero-ID key issue in the KV store
	appID := uint64(1)
	application, err := ethiqtypes.GetApplicationByID(appID)
	suite.Require().NoError(err)

	fromAddr := sdk.MustAccAddressFromBech32(application.FromAddress)

	// Fund the from address with enough aISLM to cover:
	// BurnedBeforeAmount (from app 1) is 1e18, BurnAmount is 10e18.
	// The app uses CalculateHaqqAmount which needs the accumulated amount.
	// Fund enough for the burn.
	burnCoins := sdk.NewCoins(sdk.NewCoin("aISLM", application.BurnAmount.Amount.MulRaw(2)))
	fundErr := s.network.App.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, burnCoins)
	suite.Require().NoError(fundErr)
	fundErr = s.network.App.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, fromAddr, burnCoins)
	suite.Require().NoError(fundErr)

	msgSrv := ethiqkeeper.NewMsgServerImpl(s.network.App.EthiqKeeper)

	msg := &ethiqtypes.MsgMintHaqqByApplication{
		FromAddress:   fromAddr.String(),
		ApplicationId: appID,
	}

	// First execution should succeed
	res, err := msgSrv.MintHaqqByApplication(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Second execution should fail with "already executed"
	_, err = msgSrv.MintHaqqByApplication(ctx, msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "already executed")
}
