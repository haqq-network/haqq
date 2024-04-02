package testdata

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/x/erc20/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed ERC20NoMetadata.json
	ERC20NoMetadataJSON []byte //nolint: golint

	// ERC20NoMetadataContract is the compiled erc20 contract
	ERC20NoMetadataContract evmtypes.CompiledContract

	// ERC20NoMetadataAddress is the erc20 module address
	ERC20NoMetadataAddress common.Address
)

func init() {
	ERC20NoMetadataAddress = types.ModuleAddress

	err := json.Unmarshal(ERC20NoMetadataJSON, &ERC20NoMetadataContract)
	if err != nil {
		panic(err)
	}

	if len(ERC20NoMetadataContract.Bin) == 0 {
		panic("load contract failed")
	}
}
