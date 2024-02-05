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

	RouterKey = ModuleName
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
