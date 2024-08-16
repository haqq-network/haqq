package v174

const (
	// UpgradeName is the shared upgrade plan name for mainnet and testnet
	UpgradeName = "v1.7.4"

	// VestingStretchLength defines new vesting periods length as three years in days
	VestingStretchLength = 1095

	// LockupLengthThreshold defines threshold parameter as 1767225600 is timestamp for 2026-01-01
	LockupLengthThreshold = 1767225600

	// OneDayInSeconds defines one day in seconds
	OneDayInSeconds = int64(86_400)
)
