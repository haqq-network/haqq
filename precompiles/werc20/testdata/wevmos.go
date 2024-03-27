package testdata

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed WISLM.json
	WislmJSON []byte

	// WISLMContract is the compiled contract of WISLM
	WISLMContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(WislmJSON, &WISLMContract)
	if err != nil {
		panic(err)
	}

	if len(WISLMContract.Bin) == 0 {
		panic("failed to load WISLM smart contract")
	}
}
