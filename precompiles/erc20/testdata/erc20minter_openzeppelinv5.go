// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package testdata

import (
	contractutils "github.com/haqq-network/haqq/contracts/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func LoadERC20MinterV5Contract() (evmtypes.CompiledContract, error) {
	return contractutils.LegacyLoadContractFromJSONFile("ERC20Minter_OpenZeppelinV5.json")
}
