package types

const (
	// ModuleName defines the module name
	ModuleName = "liquidvesting"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// ChainKeyPrefix is the prefix to retrieve all Chain
	DenomKeyPrefix = "Denom/value/"

	// ChainCounterKey is the prefix to store chain counter
	DenomCounterKey = "Denom/count/"

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_liquidvesting"

	RouterKey = ModuleName
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
