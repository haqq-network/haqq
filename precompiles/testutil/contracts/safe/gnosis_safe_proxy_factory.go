// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package contracts

import (
	contractutils "github.com/haqq-network/haqq/contracts/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// LoadGnosisSafeProxyFactoryContract returns the compiled GnosisSafeProxyFactory artifact used in ethiq Safe flow tests.
func LoadGnosisSafeProxyFactoryContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("GnosisSafeProxyFactory.json")
}

// LoadGnosisSafeContract returns the compiled GnosisSafe singleton artifact used in ethiq Safe flow tests.
func LoadGnosisSafeContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("GnosisSafe.json")
}
