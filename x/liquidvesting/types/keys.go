package types

const (
	// ModuleName defines the module name
	ModuleName = "liquidvesting"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// DenomKeyPrefix is the prefix to retrieve all Denom
	DenomKeyPrefix = "Denom/value/"

	// DenomCounterKey is the prefix to store denom counter
	DenomCounterKey = "Denom/count/"

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_liquidvesting"

	RouterKey = ModuleName
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
