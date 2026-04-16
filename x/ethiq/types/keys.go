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
	// TotalBurnedFromApplicationsAmountKey is the key for total burned amount
	TotalBurnedFromApplicationsAmountKey = []byte{0x02}
	// ExecutedApplicationsPrefix is the key prefix for the index of executed application IDs
	ExecutedApplicationsPrefix = []byte{0x03}
)
