package contracts

import (
	contractutils "github.com/haqq-network/haqq/contracts/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// LoadModuleAccountToucherContract loads the helper used to dirty a Cosmos module
// account in the EVM StateDB and call a precompile in the same transaction.
func LoadModuleAccountToucherContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("ModuleAccountToucher.json")
}
