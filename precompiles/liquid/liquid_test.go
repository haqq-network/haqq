package liquid_test

import (
	"github.com/haqq-network/haqq/precompiles/liquid"
)

func (s *PrecompileTestSuite) TestIsTransaction() {
	testCases := []struct {
		name   string
		method string
		expTx  bool
	}{
		{"liquidate is a transaction", liquid.LiquidateMethod, true},
		{"redeem is a transaction", liquid.RedeemMethod, true},
		{"unknown method is not a transaction", "unknown", false},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.expTx, s.precompile.IsTransaction(tc.method))
		})
	}
}

func (s *PrecompileTestSuite) TestLogger() {
	s.SetupTest()
	ctx := s.network.GetContext()
	logger := s.precompile.Logger(ctx)
	s.Require().NotNil(logger)
}
