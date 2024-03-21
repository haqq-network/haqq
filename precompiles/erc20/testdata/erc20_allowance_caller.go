package testdata

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed ERC20AllowanceCaller.json
	ERC20AllowanceCaller []byte

	// ERC20AllowanceCallerContract is the compiled contract calling the staking precompile
	ERC20AllowanceCallerContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(ERC20AllowanceCaller, &ERC20AllowanceCallerContract)
	if err != nil {
		panic(err)
	}

	if len(ERC20AllowanceCallerContract.Bin) == 0 {
		panic("failed to load smart contract that calls erc20 precompile allowance functionality")
	}
}
