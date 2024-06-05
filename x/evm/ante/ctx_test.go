package ante_test

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	evmante "github.com/haqq-network/haqq/x/evm/ante"
)

func (s *EvmAnteTestSuite) TestBuildEvmExecutionCtx() {
	n := network.New()
	ctx := evmante.BuildEvmExecutionCtx(n.GetContext())

	s.Equal(storetypes.GasConfig{}, ctx.KVGasConfig())
	s.Equal(storetypes.GasConfig{}, ctx.TransientKVGasConfig())
}
