package contracts

import (
	_ "embed"
	"encoding/json"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed compiled_contracts/haqq_testing.json
	HaqqTestingJSON []byte

	// HaqqTestingContract is the compiled dummy contract
	HaqqTestingContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(HaqqTestingJSON, &HaqqTestingContract)
	if err != nil {
		panic(err)
	}

	if len(HaqqTestingContract.Bin) == 0 {
		panic("load contract failed")
	}
}
