package contracts

import (
	contractutils "github.com/haqq-network/haqq/contracts/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// LoadPrecompileChaosContract loads the fixture that runs random programs of
// value transfers, precompile calls and reverting sub-programs.
func LoadPrecompileChaosContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("PrecompileChaos.json")
}
