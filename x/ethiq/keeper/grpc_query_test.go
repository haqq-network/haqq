package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/utils"
	ethiqkeeper "github.com/haqq-network/haqq/x/ethiq/keeper"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

func (suite *KeeperTestSuite) TestTotalBurnedGRPC() {
	testCases := []struct {
		name        string
		req         *ethiqtypes.QueryTotalBurnedRequest
		malleate    func(ctx sdk.Context)
		expRes      *ethiqtypes.QueryTotalBurnedResponse
		expErr      bool
		errContains string
	}{
		{
			name:        "fail - nil request",
			malleate:    func(_ sdk.Context) {},
			expErr:      true,
			errContains: "empty request",
		},
		{
			name:     "success - zero at start",
			req:      &ethiqtypes.QueryTotalBurnedRequest{},
			malleate: func(_ sdk.Context) {},
			expRes: &ethiqtypes.QueryTotalBurnedResponse{
				TotalBurned: sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt()),
			},
			expErr: false,
		},
		{
			name: "success - 1 ISLM after burn via ethiq module",
			req:  &ethiqtypes.QueryTotalBurnedRequest{},
			malleate: func(ctx sdk.Context) {
				_, err := s.network.App.EthiqKeeper.BurnIslmForHaqq(ctx, sdkmath.OneInt().MulRaw(1e18), s.keyring.GetAccAddr(0), s.keyring.GetAccAddr(0))
				suite.Require().NoError(err)
			},
			expRes: &ethiqtypes.QueryTotalBurnedResponse{
				TotalBurned: sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt().MulRaw(1e18)),
			},
			expErr: false,
		},
		{
			name: "success - Zero ISLM after direct burn via bank module",
			req:  &ethiqtypes.QueryTotalBurnedRequest{},
			malleate: func(ctx sdk.Context) {
				oneIslm := sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt().MulRaw(1e18))
				err := s.network.App.BankKeeper.SendCoinsFromAccountToModule(ctx, s.keyring.GetAccAddr(0), ethiqtypes.ModuleName, sdk.NewCoins(oneIslm))
				suite.Require().NoError(err)

				err = s.network.App.BankKeeper.BurnCoins(ctx, ethiqtypes.ModuleName, sdk.NewCoins(oneIslm))
				suite.Require().NoError(err)
			},
			expRes: &ethiqtypes.QueryTotalBurnedResponse{
				TotalBurned: sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt()),
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()
			tc.malleate(ctx)

			res, err := s.network.App.EthiqKeeper.TotalBurned(ctx, tc.req)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(tc.expRes.TotalBurned, res.GetTotalBurned())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCalculateGRPC() {
	testCases := []struct {
		name        string
		req         *ethiqtypes.QueryCalculateRequest
		malleate    func(ctx sdk.Context)
		expRes      *ethiqtypes.QueryCalculateResponse
		calcExpRes  bool
		expErr      bool
		errContains string
	}{
		{
			name:        "fail - nil request",
			malleate:    func(_ sdk.Context) {},
			expErr:      true,
			errContains: "empty request",
		},
		{
			name: "fail - zero amount",
			req: &ethiqtypes.QueryCalculateRequest{
				IslmAmount: "0",
			},
			malleate:    func(_ sdk.Context) {},
			expErr:      true,
			errContains: "islm_amount must be positive",
		},
		{
			name: "fail - module disabled",
			req: &ethiqtypes.QueryCalculateRequest{
				IslmAmount: "1",
			},
			malleate: func(ctx sdk.Context) {
				p := s.network.App.EthiqKeeper.GetParams(ctx)
				p.Enabled = false
				s.network.App.EthiqKeeper.SetParams(ctx, p)
			},
			expErr:      true,
			errContains: "module is disabled",
		},
		{
			name: "fail - invalid string amount",
			req: &ethiqtypes.QueryCalculateRequest{
				IslmAmount: "not-a-number",
			},
			malleate:    func(_ sdk.Context) {},
			expErr:      true,
			errContains: "invalid islm amount",
		},
		{
			name: "success - valid positive amount",
			req: &ethiqtypes.QueryCalculateRequest{
				IslmAmount: "1000000000000000000",
			},
			malleate: func(_ sdk.Context) {},
			expRes: &ethiqtypes.QueryCalculateResponse{
				EstimatedHaqqAmount: sdkmath.NewIntFromUint64(102564102564102564),
			},
			calcExpRes: true,
			expErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()
			tc.malleate(ctx)
			if tc.calcExpRes && tc.req != nil {
				islmAmount, ok := sdkmath.NewIntFromString(tc.req.IslmAmount)
				suite.Require().True(ok)
				expAmt, expErr := s.network.App.EthiqKeeper.CalculateHaqqCoinsToMint(ctx, islmAmount)
				suite.Require().NoError(expErr)
				tc.expRes = &ethiqtypes.QueryCalculateResponse{EstimatedHaqqAmount: expAmt}
			}

			res, err := s.network.App.EthiqKeeper.Calculate(ctx, tc.req)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(tc.expRes.EstimatedHaqqAmount, res.EstimatedHaqqAmount)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCalculateForApplicationsGRPC() {
	testCases := []struct {
		name        string
		req         *ethiqtypes.QueryCalculateForApplicationRequest
		expRes      *ethiqtypes.QueryCalculateForApplicationResponse
		calcExpRes  bool
		expErr      bool
		errContains string
	}{
		{
			name:        "fail - nil request",
			expErr:      true,
			errContains: "empty request",
		},
		{
			name: "fail - application not found",
			req: &ethiqtypes.QueryCalculateForApplicationRequest{
				ApplicationId: 9999,
			},
			expErr:      true,
			errContains: "application not found",
		},
		{
			name: "success - application id = 0",
			req: &ethiqtypes.QueryCalculateForApplicationRequest{
				ApplicationId: 0,
			},
			expRes: &ethiqtypes.QueryCalculateForApplicationResponse{
				EstimatedHaqqAmount: sdkmath.NewIntFromUint64(333333333333333333),
			},
			calcExpRes: true,
			expErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()
			if tc.calcExpRes && tc.req != nil {
				app, appErr := ethiqtypes.GetApplicationByID(tc.req.ApplicationId)
				suite.Require().NoError(appErr)
				expAmt, expErr := ethiqkeeper.CalculateHaqqAmount(app.BurnedBeforeAmount.Amount, app.BurnAmount.Amount)
				suite.Require().NoError(expErr)
				tc.expRes = &ethiqtypes.QueryCalculateForApplicationResponse{EstimatedHaqqAmount: expAmt}
			}

			res, err := s.network.App.EthiqKeeper.CalculateForApplication(ctx, tc.req)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(tc.expRes.EstimatedHaqqAmount, res.EstimatedHaqqAmount)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetApplicationsGRPC() {
	testCases := []struct {
		name        string
		req         *ethiqtypes.QueryGetApplicationsRequest
		expLen      int
		expTotal    uint64
		expErr      bool
		errContains string
	}{
		{
			name:        "fail - nil request",
			req:         nil,
			expErr:      true,
			errContains: "empty request",
		},
		{
			name:     "success - with nil pagination",
			req:      &ethiqtypes.QueryGetApplicationsRequest{},
			expLen:   int(query.DefaultLimit), //nolint: gosec // G115
			expTotal: 0,
			expErr:   false,
		},
		{
			name: "success - with limited pagination, no count total",
			req: &ethiqtypes.QueryGetApplicationsRequest{
				Pagination: &query.PageRequest{
					Limit:  10,
					Offset: 0,
				},
			},
			expLen:   10,
			expTotal: 0,
			expErr:   false,
		},
		{
			name: "success - with limited pagination, with count total",
			req: &ethiqtypes.QueryGetApplicationsRequest{
				Pagination: &query.PageRequest{
					Limit:      10,
					Offset:     0,
					CountTotal: true,
				},
			},
			expLen:   10,
			expTotal: ethiqtypes.TotalNumberOfApplications(),
			expErr:   false,
		},
		{
			name: "success - default pagination",
			req: &ethiqtypes.QueryGetApplicationsRequest{
				Pagination: &query.PageRequest{},
			},
			expLen:   int(query.DefaultLimit), //nolint: gosec // G115
			expTotal: 0,
			expErr:   false,
		},
		{
			name: "success - offset beyond total",
			req: &ethiqtypes.QueryGetApplicationsRequest{
				Pagination: &query.PageRequest{
					Limit:  10,
					Offset: ethiqtypes.TotalNumberOfApplications() + 100,
				},
			},
			expLen:   0,
			expTotal: 0,
			expErr:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()

			res, err := s.network.App.EthiqKeeper.GetApplications(ctx, tc.req)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotNil(res.Pagination)
				suite.Require().Equal(tc.expLen, len(res.Applications))
				suite.Require().Equal(tc.expTotal, res.Pagination.Total)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetSendersApplicationsGRPC() {
	// Use a known sender from the waitlist
	knownSender := "haqq13x3h3t9fqc69er64jmf3za5wuz9fkd2zpgl737"

	testCases := []struct {
		name        string
		req         *ethiqtypes.QueryGetSendersApplicationsRequest
		expLen      int
		expTotal    uint64
		expErr      bool
		errContains string
	}{
		{
			name:        "fail - nil request",
			req:         nil,
			expErr:      true,
			errContains: "empty request",
		},
		{
			name: "success - known sender, with nil pagination",
			req: &ethiqtypes.QueryGetSendersApplicationsRequest{
				SenderAddress: knownSender,
			},
			expLen:   int(ethiqtypes.TotalNumberOfApplicationsBySender(knownSender)), //nolint: gosec // G115
			expTotal: 0,
			expErr:   false,
		},
		{
			name: "success - known sender, with limited pagination, no count total",
			req: &ethiqtypes.QueryGetSendersApplicationsRequest{
				SenderAddress: knownSender,
				Pagination: &query.PageRequest{
					Limit:  5,
					Offset: 0,
				},
			},
			expLen:   5,
			expTotal: 0,
			expErr:   false,
		},
		{
			name: "success - known sender, with limited pagination, with count total",
			req: &ethiqtypes.QueryGetSendersApplicationsRequest{
				SenderAddress: knownSender,
				Pagination: &query.PageRequest{
					Limit:      5,
					Offset:     0,
					CountTotal: true,
				},
			},
			expLen:   5,
			expTotal: ethiqtypes.TotalNumberOfApplicationsBySender(knownSender),
			expErr:   false,
		},
		{
			name: "success - known sender, default pagination",
			req: &ethiqtypes.QueryGetSendersApplicationsRequest{
				SenderAddress: knownSender,
				Pagination:    &query.PageRequest{},
			},
			expLen:   int(ethiqtypes.TotalNumberOfApplicationsBySender(knownSender)), //nolint: gosec // G115
			expTotal: 0,
			expErr:   false,
		},
		{
			name: "success - unknown sender returns empty, with count total",
			req: &ethiqtypes.QueryGetSendersApplicationsRequest{
				SenderAddress: "haqq1zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
				Pagination: &query.PageRequest{
					Limit:      10,
					Offset:     0,
					CountTotal: true,
				},
			},
			expLen:   0,
			expTotal: 0,
			expErr:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()

			res, err := s.network.App.EthiqKeeper.GetSendersApplications(ctx, tc.req)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotNil(res.Pagination)
				suite.Require().Equal(tc.expLen, len(res.Applications))
				suite.Require().Equal(tc.expTotal, res.Pagination.Total)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestParamsGRPC() {
	testCases := []struct {
		name        string
		req         *ethiqtypes.QueryParamsRequest
		expErr      bool
		errContains string
	}{
		{
			name:        "fail - nil request",
			req:         nil,
			expErr:      true,
			errContains: "empty request",
		},
		{
			name:   "success - returns default params",
			req:    &ethiqtypes.QueryParamsRequest{},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()

			res, err := s.network.App.EthiqKeeper.Params(ctx, tc.req)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				defaultParams := ethiqtypes.DefaultParams()
				suite.Require().Equal(defaultParams.Enabled, res.Params.Enabled)
			}
		})
	}
}
