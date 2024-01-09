package types

const (
	// ModuleName defines the module name
	ModuleName = "liquidvesting"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_liquidvesting"
)

var (
	ParamsKey = []byte("p_liquidvesting")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
