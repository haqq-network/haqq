package keeper_test

import (
	sdkmath "cosmossdk.io/math"

	"github.com/haqq-network/haqq/x/ethiq/keeper"
)

func (us *UnitTestSuite) TestCalculate() {
	testCases := []struct {
		name            string
		alreadyBurntAmt func() sdkmath.Int
		burnAmt         func() sdkmath.Int
		mintAmt         func() sdkmath.Int
		expErr          string
		success         bool
	}{
		{
			"success - level 1, 3m burn, 1m mint",
			sdkmath.ZeroInt,
			func() sdkmath.Int {
				// burn 3m ISLM within single price level 1:3
				amt, _ := sdkmath.NewIntFromString("3000000000000000000000000")
				return amt
			},
			func() sdkmath.Int {
				// 1m HAQQ tokens to be minted
				amt, _ := sdkmath.NewIntFromString("1000000000000000000000000")
				return amt
			},
			"",
			true,
		},
		{
			"success - level 1, full burn",
			sdkmath.ZeroInt,
			func() sdkmath.Int {
				// birn full amount of price level, 40m ISLM
				amt, _ := sdkmath.NewIntFromString("40000000000000000000000000")
				return amt
			},
			func() sdkmath.Int {
				// receive 13333333333333333333333333 aHAQQ
				// as top border of price level threshold doesn't included
				// for burn available 39999999999999999999999999 aISLM
				// the rest amount for the next level is less than price ratio
				amt, _ := sdkmath.NewIntFromString("13333333333333333333333333")
				return amt
			},
			"",
			true,
		},
		{
			"success - part level 1, part level 2, 7m burn, 2m mint",
			func() sdkmath.Int {
				// already burnt 36999999999999999999999999 aISLM
				// the rest amount on this price level is 3m ISLM
				amt, _ := sdkmath.NewIntFromString("36999999999999999999999999")
				return amt
			},
			func() sdkmath.Int {
				// burn 7m ISLM, 3m on current level, 4m on the next one
				amt, _ := sdkmath.NewIntFromString("7000000000000000000000000")
				return amt
			},
			func() sdkmath.Int {
				// receive 2m HAQQ
				// as 1m received on price level 1:3 by burning 3m
				// and 1m on level 1:4 by burning 4m
				amt, _ := sdkmath.NewIntFromString("2000000000000000000000000")
				return amt
			},
			"",
			true,
		},
		{
			"success - level 2, full burn",
			func() sdkmath.Int {
				// already burnt 39999999999999999999999999 aISLM
				// full level 1 is burnt, full level 2 is available (20m ISLM)
				amt, _ := sdkmath.NewIntFromString("39999999999999999999999999")
				return amt
			},
			func() sdkmath.Int {
				// birn full amount of price level 2, 20m ISLM
				amt, _ := sdkmath.NewIntFromString("20000000000000000000000000")
				return amt
			},
			func() sdkmath.Int {
				// receive 5m HAQQ
				// level 2 rate 1:4
				// for burn available 20m ISLM
				amt, _ := sdkmath.NewIntFromString("5000000000000000000000000")
				return amt
			},
			"",
			true,
		},
		{
			"success - part level 11, full level 12, part level 13, 153+m burn, 2m mint",
			func() sdkmath.Int {
				// select level 11, already burnt amount 160m ISLM, rate 1:13,5
				amt, _ := sdkmath.NewIntFromString("160000000000000000000000000")
				return amt
			},
			func() sdkmath.Int {
				// burn huge amount to cover 3 different price levels
				// rate 1:13,5 for 13749999999999999999999999 aISLM
				// rate 1:14.75 for 16250000000000000000000000 aISLM
				// rate 1:16 for 5345678998723431876345322 aISLM
				// total: 35345678998723431876345321 aISLM
				amt, _ := sdkmath.NewIntFromString("35345678998723431876345321")
				return amt
			},
			func() sdkmath.Int {
				// mint huge amount within 3 different price levels
				// rate 1:13,5 gives 1018518518518518518518518 aHAQQ
				// rate 1:14.75 gives 1101694915254237288135593 aHAQQ
				// rate 1:16 gives 334104937420214492271582 aHAQQ
				// total: 2454318371192970298925693 aHAQQ
				amt, _ := sdkmath.NewIntFromString("2454318371192970298925693")
				return amt
			},
			"",
			true,
		},
	}

	for _, tc := range testCases {
		us.Run(tc.name, func() {
			res, err := keeper.CalculateHaqqAmount(tc.alreadyBurntAmt(), tc.burnAmt())
			if !tc.success {
				us.Require().ErrorContains(err, tc.expErr)
				us.Require().Equal(sdkmath.Int{}, res)
				return
			}

			us.Require().NoError(err)
			us.Require().Equal(tc.mintAmt().String(), res.String())
		})
	}
}
