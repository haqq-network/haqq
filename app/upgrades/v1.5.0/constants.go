package v150

const (
	// UpgradeName is the shared upgrade plan name for mainnet and testnet
	UpgradeName = "v1.5.0"

	// internal constants
	cliffPeriod        = int64(15552000) // 6 months in seconds
	unlockPeriod       = int64(2592000)  // 1 month in seconds
	historyStateHeight = 100000
	threshold          = "10000000000000000000" // 10 ISLM for tests
	vestingContract    = "0x40a3e24b85D32f3f68Ee9e126B8dD9dBC2D301Eb"
)
