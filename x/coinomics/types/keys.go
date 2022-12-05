package types

var ParamsKey = []byte{0x00}

// constants
const (
	// module name
	ModuleName = "coinomics"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for message routing
	RouterKey = ModuleName
)

// prefix bytes for the inflation persistent store
const (
	prefixInflation = iota + 1
	prefixEra
	prefixEraStartedAtBlock
	prefixEraTargetMint
	prefixEraClosingSupply
	prefixMaxSupply
)

// KVStore key prefixes
var (
	KeyPrefixInflation         = []byte{prefixInflation}
	KeyPrefixEra               = []byte{prefixEra}
	KeyPrefixEraStartedAtBlock = []byte{prefixEraStartedAtBlock}
	KetPrefixEraTargetMint     = []byte{prefixEraTargetMint}
	KeyPrefixEraClosingSupply  = []byte{prefixEraClosingSupply}
	KeyPrefixMaxSupply         = []byte{prefixMaxSupply}
)
