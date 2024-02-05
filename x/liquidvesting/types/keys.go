package types

const (
	// ModuleName defines the module name
	ModuleName = "liquidvesting"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	RouterKey = ModuleName
)

const (
	prefixDenom = iota + 1
	denomCounter
)

var (
	DenomKeyPrefix  = []byte{prefixDenom}
	DenomCounterKey = []byte{denomCounter}
)
