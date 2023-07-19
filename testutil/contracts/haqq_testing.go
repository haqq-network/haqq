package contracts

import (
	_ "embed"
	"encoding/json"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

var (
	//go:embed compiled_contracts/haqq_testing.json
	HaqqTestingJSON []byte // nolint: golint

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
