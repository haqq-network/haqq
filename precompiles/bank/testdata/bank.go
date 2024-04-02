package testdata

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed BankCaller.json
	BankCallerJSON []byte

	// BankCallerContract is the compiled contract of BankCaller.sol
	BankCallerContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(BankCallerJSON, &BankCallerContract)
	if err != nil {
		panic(err)
	}

	if len(BankCallerContract.Bin) == 0 {
		panic("failed to load BankCaller smart contract")
	}
}
