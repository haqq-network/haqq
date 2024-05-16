package keeper_test

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/x/epochs/types"
)

func (suite *KeeperTestSuite) TestEpochInfo() {
	var (
		req         *types.QueryEpochsInfoRequest
		expResponse *types.QueryEpochsInfoResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"default EpochInfos",
			func() {
				req = &types.QueryEpochsInfoRequest{}

				day := types.EpochInfo{
					Identifier:              types.DayEpochID,
					StartTime:               time.Time{},
					Duration:                time.Hour * 24,
					CurrentEpoch:            0,
					CurrentEpochStartHeight: 1,
					CurrentEpochStartTime:   time.Time{},
					EpochCountingStarted:    false,
				}
				day.StartTime = suite.ctx.BlockTime()
				day.CurrentEpochStartHeight = suite.ctx.BlockHeight()

				week := types.EpochInfo{
					Identifier:              types.WeekEpochID,
					StartTime:               time.Time{},
					Duration:                time.Hour * 24 * 7,
					CurrentEpoch:            0,
					CurrentEpochStartHeight: 1,
					CurrentEpochStartTime:   time.Time{},
					EpochCountingStarted:    false,
				}
				week.StartTime = suite.ctx.BlockTime()
				week.CurrentEpochStartHeight = suite.ctx.BlockHeight()

				expResponse = &types.QueryEpochsInfoResponse{
					Epochs: []types.EpochInfo{day, week},
					Pagination: &query.PageResponse{
						NextKey: nil,
						Total:   uint64(2),
					},
				}
			},
			true,
		},
		{
			"set epoch info",
			func() {
				day := types.EpochInfo{
					Identifier:              types.DayEpochID,
					StartTime:               time.Time{},
					Duration:                time.Hour * 24,
					CurrentEpoch:            0,
					CurrentEpochStartHeight: 1,
					CurrentEpochStartTime:   time.Time{},
					EpochCountingStarted:    false,
				}
				day.StartTime = suite.ctx.BlockTime()
				day.CurrentEpochStartHeight = suite.ctx.BlockHeight()

				week := types.EpochInfo{
					Identifier:              types.WeekEpochID,
					StartTime:               time.Time{},
					Duration:                time.Hour * 24 * 7,
					CurrentEpoch:            0,
					CurrentEpochStartHeight: 1,
					CurrentEpochStartTime:   time.Time{},
					EpochCountingStarted:    false,
				}
				week.StartTime = suite.ctx.BlockTime()
				week.CurrentEpochStartHeight = suite.ctx.BlockHeight()

				quarter := types.EpochInfo{
					Identifier:              "quarter",
					StartTime:               time.Time{},
					Duration:                time.Hour * 24 * 7 * 13,
					CurrentEpoch:            0,
					CurrentEpochStartHeight: 1,
					CurrentEpochStartTime:   time.Time{},
					EpochCountingStarted:    false,
				}
				quarter.StartTime = suite.ctx.BlockTime()
				quarter.CurrentEpochStartHeight = suite.ctx.BlockHeight()
				suite.app.EpochsKeeper.SetEpochInfo(suite.ctx, quarter)
				suite.Commit()

				req = &types.QueryEpochsInfoRequest{}
				expResponse = &types.QueryEpochsInfoResponse{
					Epochs: []types.EpochInfo{day, quarter, week},
					Pagination: &query.PageResponse{
						NextKey: nil,
						Total:   uint64(3),
					},
				}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()

			res, err := suite.queryClient.EpochInfos(ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expResponse, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCurrentEpoch() {
	var (
		req         *types.QueryCurrentEpochRequest
		expResponse *types.QueryCurrentEpochResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"unknown identifier",
			func() {
				defaultCurrentEpoch := int64(0)
				req = &types.QueryCurrentEpochRequest{Identifier: "second"}
				expResponse = &types.QueryCurrentEpochResponse{
					CurrentEpoch: defaultCurrentEpoch,
				}
			},
			false,
		},
		{
			"week - default currentEpoch",
			func() {
				defaultCurrentEpoch := int64(0)
				req = &types.QueryCurrentEpochRequest{Identifier: types.WeekEpochID}
				expResponse = &types.QueryCurrentEpochResponse{
					CurrentEpoch: defaultCurrentEpoch,
				}
			},
			true,
		},
		{
			"day - default currentEpoch",
			func() {
				defaultCurrentEpoch := int64(0)
				req = &types.QueryCurrentEpochRequest{Identifier: types.DayEpochID}
				expResponse = &types.QueryCurrentEpochResponse{
					CurrentEpoch: defaultCurrentEpoch,
				}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()

			res, err := suite.queryClient.CurrentEpoch(ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expResponse, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
