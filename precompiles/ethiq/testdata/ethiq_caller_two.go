package testdata

import (
	contractutils "github.com/haqq-network/haqq/contracts/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func LoadEthiqCallerTwoContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("EthiqCallerTwo.json")
}
