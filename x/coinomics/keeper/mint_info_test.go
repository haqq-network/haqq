package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestSetGetPrevBlockTs() {
	expEra := math.NewInt(100)

	testCases := []struct {
		name     string
		malleate func()
		ok       bool
	}{
		{
			"default prevblockts",
			func() {},
			false,
		},
		{
			"prevblockts set",
			func() {
				s.network.App.CoinomicsKeeper.SetPrevBlockTS(s.network.GetContext(), expEra)
			},
			true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset

			tc.malleate()

			prevBlockTS := s.network.App.CoinomicsKeeper.GetPrevBlockTS(s.network.GetContext())
			if tc.ok {
				s.Require().Equal(expEra.String(), prevBlockTS.String(), tc.name)
			} else {
				s.Require().Equal(s.network.GetContext().BlockTime().UnixMilli(), prevBlockTS.Int64(), tc.name)
			}
		})
	}
}

func (s *KeeperTestSuite) TestSetGetMaxSupply() {
	defaultMaxSupply := sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(100_000_000_000, 18)}
	expMaxSupply := sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(1337, 18)}

	testCases := []struct {
		name     string
		malleate func()
		ok       bool
	}{
		{
			"default MaxSupply",
			func() {},
			false,
		},
		{
			"MaxSupply set",
			func() {
				s.network.App.CoinomicsKeeper.SetMaxSupply(s.network.GetContext(), expMaxSupply)
			},
			true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset

			tc.malleate()

			maxSupply := s.network.App.CoinomicsKeeper.GetMaxSupply(s.network.GetContext())

			if tc.ok {
				s.Require().Equal(expMaxSupply, maxSupply, tc.name)
			} else {
				s.Require().Equal(defaultMaxSupply, maxSupply, tc.name)
			}
		})
	}
}
