package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

func (suite *KeeperTestSuite) TestCreateDenom() {
	var ctx sdk.Context

	testCases := []struct {
		name string
		run  func()
	}{
		{
			"creates denom with correct fields",
			func() {
				startTime := int64(1_000_000)
				periods := sdkvesting.Periods{
					{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
				}

				denom, err := suite.network.App.LiquidVestingKeeper.CreateDenom(ctx, utils.BaseDenom, startTime, periods)
				suite.Require().NoError(err)

				suite.Require().Equal(types.DenomBaseNameFromID(0), denom.BaseDenom)
				suite.Require().Equal(types.DenomDisplayNameFromID(0), denom.DisplayDenom)
				suite.Require().Equal(utils.BaseDenom, denom.OriginalDenom)
				suite.Require().Equal(time.Unix(startTime, 0), denom.StartTime)
				suite.Require().Equal(time.Unix(startTime+periods.TotalLength(), 0), denom.EndTime)
				suite.Require().Equal(periods, denom.LockupPeriods)
			},
		},
		{
			"increments denom counter on create",
			func() {
				periods := sdkvesting.Periods{
					{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
				}

				counterBefore := suite.network.App.LiquidVestingKeeper.GetDenomCounter(ctx)
				suite.Require().Equal(uint64(0), counterBefore)

				_, err := suite.network.App.LiquidVestingKeeper.CreateDenom(ctx, utils.BaseDenom, 1_000_000, periods)
				suite.Require().NoError(err)

				counterAfter := suite.network.App.LiquidVestingKeeper.GetDenomCounter(ctx)
				suite.Require().Equal(uint64(1), counterAfter)
			},
		},
		{
			"multiple creates increment counter and assign correct base denoms",
			func() {
				periods := sdkvesting.Periods{
					{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
				}

				denom0, err := suite.network.App.LiquidVestingKeeper.CreateDenom(ctx, utils.BaseDenom, 1_000_000, periods)
				suite.Require().NoError(err)
				suite.Require().Equal(types.DenomBaseNameFromID(0), denom0.BaseDenom)
				suite.Require().Equal(types.DenomDisplayNameFromID(0), denom0.DisplayDenom)

				denom1, err := suite.network.App.LiquidVestingKeeper.CreateDenom(ctx, utils.BaseDenom, 2_000_000, periods)
				suite.Require().NoError(err)
				suite.Require().Equal(types.DenomBaseNameFromID(1), denom1.BaseDenom)
				suite.Require().Equal(types.DenomDisplayNameFromID(1), denom1.DisplayDenom)

				denom2, err := suite.network.App.LiquidVestingKeeper.CreateDenom(ctx, utils.BaseDenom, 3_000_000, periods)
				suite.Require().NoError(err)
				suite.Require().Equal(types.DenomBaseNameFromID(2), denom2.BaseDenom)
				suite.Require().Equal(types.DenomDisplayNameFromID(2), denom2.DisplayDenom)

				suite.Require().Equal(uint64(3), suite.network.App.LiquidVestingKeeper.GetDenomCounter(ctx))
			},
		},
		{
			"start and end time are computed correctly from periods",
			func() {
				startTime := int64(500_000)
				periods := sdkvesting.Periods{
					{Length: 86400, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 500_000))},
					{Length: 86400, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 500_000))},
				}

				denom, err := suite.network.App.LiquidVestingKeeper.CreateDenom(ctx, utils.BaseDenom, startTime, periods)
				suite.Require().NoError(err)

				expectedStart := time.Unix(startTime, 0)
				expectedEnd := time.Unix(startTime+periods.TotalLength(), 0)

				suite.Require().Equal(expectedStart, denom.StartTime)
				suite.Require().Equal(expectedEnd, denom.EndTime)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()
			tc.run()
		})
	}
}

func (suite *KeeperTestSuite) TestSetAndGetDenom() {
	var ctx sdk.Context

	testCases := []struct {
		name string
		run  func()
	}{
		{
			"set denom and get it back with matching fields",
			func() {
				periods := sdkvesting.Periods{
					{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
				}
				startTime := int64(1_000_000)
				denom := types.Denom{
					BaseDenom:     types.DenomBaseNameFromID(0),
					DisplayDenom:  types.DenomDisplayNameFromID(0),
					OriginalDenom: utils.BaseDenom,
					StartTime:     time.Unix(startTime, 0).UTC(),
					EndTime:       time.Unix(startTime+periods.TotalLength(), 0).UTC(),
					LockupPeriods: periods,
				}

				suite.network.App.LiquidVestingKeeper.SetDenom(ctx, denom)

				got, found := suite.network.App.LiquidVestingKeeper.GetDenom(ctx, denom.BaseDenom)
				suite.Require().True(found)
				suite.Require().Equal(denom.BaseDenom, got.BaseDenom)
				suite.Require().Equal(denom.DisplayDenom, got.DisplayDenom)
				suite.Require().Equal(denom.OriginalDenom, got.OriginalDenom)
				suite.Require().True(denom.StartTime.Equal(got.StartTime))
				suite.Require().True(denom.EndTime.Equal(got.EndTime))
				suite.Require().Equal(denom.LockupPeriods, got.LockupPeriods)
			},
		},
		{
			"get non-existent denom returns found=false",
			func() {
				_, found := suite.network.App.LiquidVestingKeeper.GetDenom(ctx, types.DenomBaseNameFromID(999))
				suite.Require().False(found)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()
			tc.run()
		})
	}
}

func (suite *KeeperTestSuite) TestDeleteDenom() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	periods := sdkvesting.Periods{
		{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
	}
	startTime := int64(1_000_000)
	denom := types.Denom{
		BaseDenom:     types.DenomBaseNameFromID(0),
		DisplayDenom:  types.DenomDisplayNameFromID(0),
		OriginalDenom: utils.BaseDenom,
		StartTime:     time.Unix(startTime, 0),
		EndTime:       time.Unix(startTime+periods.TotalLength(), 0),
		LockupPeriods: periods,
	}

	suite.network.App.LiquidVestingKeeper.SetDenom(ctx, denom)

	_, found := suite.network.App.LiquidVestingKeeper.GetDenom(ctx, denom.BaseDenom)
	suite.Require().True(found)

	suite.network.App.LiquidVestingKeeper.DeleteDenom(ctx, denom.BaseDenom)

	_, found = suite.network.App.LiquidVestingKeeper.GetDenom(ctx, denom.BaseDenom)
	suite.Require().False(found)
}

func (suite *KeeperTestSuite) TestUpdateDenomPeriods() {
	var ctx sdk.Context

	testCases := []struct {
		name string
		run  func()
	}{
		{
			"update periods of existing denom",
			func() {
				periods := sdkvesting.Periods{
					{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
				}
				startTime := int64(1_000_000)
				denom := types.Denom{
					BaseDenom:     types.DenomBaseNameFromID(0),
					DisplayDenom:  types.DenomDisplayNameFromID(0),
					OriginalDenom: utils.BaseDenom,
					StartTime:     time.Unix(startTime, 0),
					EndTime:       time.Unix(startTime+periods.TotalLength(), 0),
					LockupPeriods: periods,
				}
				suite.network.App.LiquidVestingKeeper.SetDenom(ctx, denom)

				newPeriods := sdkvesting.Periods{
					{Length: 200000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 500_000))},
					{Length: 200000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 500_000))},
				}

				err := suite.network.App.LiquidVestingKeeper.UpdateDenomPeriods(ctx, denom.BaseDenom, newPeriods)
				suite.Require().NoError(err)

				got, found := suite.network.App.LiquidVestingKeeper.GetDenom(ctx, denom.BaseDenom)
				suite.Require().True(found)
				suite.Require().Equal(newPeriods, got.LockupPeriods)
			},
		},
		{
			"update non-existent denom returns ErrDenomNotFound",
			func() {
				newPeriods := sdkvesting.Periods{
					{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
				}

				err := suite.network.App.LiquidVestingKeeper.UpdateDenomPeriods(ctx, types.DenomBaseNameFromID(999), newPeriods)
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, types.ErrDenomNotFound)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()
			tc.run()
		})
	}
}

func (suite *KeeperTestSuite) TestGetAllDenoms() {
	var ctx sdk.Context

	testCases := []struct {
		name string
		run  func()
	}{
		{
			"empty store returns empty slice",
			func() {
				all := suite.network.App.LiquidVestingKeeper.GetAllDenoms(ctx)
				suite.Require().Empty(all)
			},
		},
		{
			"returns all denoms after setting 3",
			func() {
				periods := sdkvesting.Periods{
					{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
				}
				startTime := int64(1_000_000)

				for i := range 3 {
					denom := types.Denom{
						BaseDenom:     types.DenomBaseNameFromID(uint64(i)),    //nolint: gosec // G115
						DisplayDenom:  types.DenomDisplayNameFromID(uint64(i)), //nolint: gosec // G115
						OriginalDenom: utils.BaseDenom,
						StartTime:     time.Unix(startTime, 0),
						EndTime:       time.Unix(startTime+periods.TotalLength(), 0),
						LockupPeriods: periods,
					}
					suite.network.App.LiquidVestingKeeper.SetDenom(ctx, denom)
				}

				all := suite.network.App.LiquidVestingKeeper.GetAllDenoms(ctx)
				suite.Require().Len(all, 3)

				baseNames := make(map[string]bool)
				for _, d := range all {
					baseNames[d.BaseDenom] = true
				}
				suite.Require().True(baseNames[types.DenomBaseNameFromID(0)])
				suite.Require().True(baseNames[types.DenomBaseNameFromID(1)])
				suite.Require().True(baseNames[types.DenomBaseNameFromID(2)])
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()
			tc.run()
		})
	}
}

func (suite *KeeperTestSuite) TestDenomCounter() {
	var ctx sdk.Context

	testCases := []struct {
		name string
		run  func()
	}{
		{
			"default counter is 0",
			func() {
				counter := suite.network.App.LiquidVestingKeeper.GetDenomCounter(ctx)
				suite.Require().Equal(uint64(0), counter)
			},
		},
		{
			"set counter to 5 and get returns 5",
			func() {
				suite.network.App.LiquidVestingKeeper.SetDenomCounter(ctx, 5)
				counter := suite.network.App.LiquidVestingKeeper.GetDenomCounter(ctx)
				suite.Require().Equal(uint64(5), counter)
			},
		},
		{
			"CreateDenom increments counter",
			func() {
				periods := sdkvesting.Periods{
					{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
				}

				suite.Require().Equal(uint64(0), suite.network.App.LiquidVestingKeeper.GetDenomCounter(ctx))

				_, err := suite.network.App.LiquidVestingKeeper.CreateDenom(ctx, utils.BaseDenom, 1_000_000, periods)
				suite.Require().NoError(err)
				suite.Require().Equal(uint64(1), suite.network.App.LiquidVestingKeeper.GetDenomCounter(ctx))

				_, err = suite.network.App.LiquidVestingKeeper.CreateDenom(ctx, utils.BaseDenom, 2_000_000, periods)
				suite.Require().NoError(err)
				suite.Require().Equal(uint64(2), suite.network.App.LiquidVestingKeeper.GetDenomCounter(ctx))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()
			tc.run()
		})
	}
}

func (suite *KeeperTestSuite) TestIterateDenoms() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	periods := sdkvesting.Periods{
		{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
	}
	startTime := int64(1_000_000)

	for i := range 3 {
		denom := types.Denom{
			BaseDenom:     types.DenomBaseNameFromID(uint64(i)),    //nolint: gosec // G115
			DisplayDenom:  types.DenomDisplayNameFromID(uint64(i)), //nolint: gosec // G115
			OriginalDenom: utils.BaseDenom,
			StartTime:     time.Unix(startTime, 0),
			EndTime:       time.Unix(startTime+periods.TotalLength(), 0),
			LockupPeriods: periods,
		}
		suite.network.App.LiquidVestingKeeper.SetDenom(ctx, denom)
	}

	suite.Run("iterate and collect all denoms", func() {
		var collected []types.Denom
		suite.network.App.LiquidVestingKeeper.IterateDenoms(ctx, func(denom types.Denom) bool {
			collected = append(collected, denom)
			return false
		})

		suite.Require().Len(collected, 3)
	})

	suite.Run("early stop with callback returning true", func() {
		var collected []types.Denom
		suite.network.App.LiquidVestingKeeper.IterateDenoms(ctx, func(denom types.Denom) bool {
			collected = append(collected, denom)
			return true // stop after first
		})

		suite.Require().Len(collected, 1)
	})
}
