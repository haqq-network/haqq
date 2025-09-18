// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ante_test

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	evmante "github.com/haqq-network/haqq/x/evm/ante"
)

func (suite *EvmAnteTestSuite) TestBuildEvmExecutionCtx() {
	nw := network.New()

	ctx := evmante.BuildEvmExecutionCtx(nw.GetContext())

	suite.Equal(storetypes.GasConfig{}, ctx.KVGasConfig())
	suite.Equal(storetypes.GasConfig{}, ctx.TransientKVGasConfig())
}
