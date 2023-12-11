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
	prefixPrevBlockTS = iota + 1
	prefixMaxSupply
)

// KVStore key prefixes
var (
	KeyPrefixPrevBlockTS = []byte{prefixPrevBlockTS}
	KeyPrefixMaxSupply   = []byte{prefixMaxSupply}
)
