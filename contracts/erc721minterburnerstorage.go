package contracts

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed compiled_contracts/ERC721MinterBurnerStorage.json
	erc721MinterBurnerStorageJSON []byte

	// ERC20BurnableContract is the compiled ERC20Burnable contract
	ERC721MinterBurnerStorageContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(erc721MinterBurnerStorageJSON, &ERC721MinterBurnerStorageContract)
	if err != nil {
		panic(err)
	}
}
