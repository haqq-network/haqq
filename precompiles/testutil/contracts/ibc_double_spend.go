package contracts

import (
	contractutils "github.com/haqq-network/haqq/contracts/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// LoadIbcDoubleSpendContract loads the fixture that dirties a native
// ERC20 balance slot before calling the ICS20 precompile transfer.
func LoadIbcDoubleSpendContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("IbcDoubleSpend.json")
}
