// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package contracts

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed compiled_contracts/WISLM.json
	WISLMJSON []byte

	// WISLMContract is the compiled contract of WISLM
	WISLMContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(WISLMJSON, &WISLMContract)
	if err != nil {
		panic(err)
	}

	if len(WISLMContract.Bin) == 0 {
		panic("failed to load WISLM smart contract")
	}
}
