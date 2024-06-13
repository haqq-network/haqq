package contracts

import (
	_ "embed" // embed compiled smart contract
	"encoding/json"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	//go:embed compiled_contracts/CommunityApprovalCertificates.json
	communityApprovalCertificatesJSON []byte

	// ERC20BurnableContract is the compiled ERC20Burnable contract
	CommunityApprovalCertificatesContract evmtypes.CompiledContract
)

func init() {
	err := json.Unmarshal(communityApprovalCertificatesJSON, &CommunityApprovalCertificatesContract)
	if err != nil {
		panic(err)
	}
}
