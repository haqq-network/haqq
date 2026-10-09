package contracts

import (
	contractutils "github.com/haqq-network/haqq/contracts/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func LoadUcdaoForwarderContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("UcdaoForwarder.json")
}
