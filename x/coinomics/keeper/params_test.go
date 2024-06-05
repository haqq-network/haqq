package keeper_test

import (
	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (s *KeeperTestSuite) TestParams() {
	params := s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())
	expParams := types.DefaultParams()

	s.Require().Equal(expParams, params)

	s.network.App.CoinomicsKeeper.SetParams(s.network.GetContext(), params)
	newParams := s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())
	s.Require().Equal(newParams, params)
}
