package contracts

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed compiled_contracts/ERC1967Proxy.json
	ERC1967ProxyJSON []byte

	// ERC20BurnableContract is the compiled ERC20Burnable contract
	ERC1967ProxyContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(ERC1967ProxyJSON, &ERC1967ProxyContract)
	if err != nil {
		panic(err)
	}
}
