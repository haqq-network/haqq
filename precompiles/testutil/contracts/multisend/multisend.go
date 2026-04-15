// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package contracts

import (
	contractutils "github.com/haqq-network/haqq/contracts/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// LoadMultiSendContract returns the compiled MultiSend artifact used in batch transaction tests.
func LoadMultiSendContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("MultiSend.json")
}
