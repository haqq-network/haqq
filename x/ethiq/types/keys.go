package types

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "ethiq"

	// StoreKey is the store key string for ethiq
	StoreKey = ModuleName

	// RouterKey is the message route for ethiq
	RouterKey = ModuleName
)

var (
	// ParamsKey is the key for ethiq module params
	ParamsKey = []byte{0x00}

	// TotalBurnedAmountKey is the key for total burned amount
	TotalBurnedAmountKey = []byte{0x01}
)
