package v150

const (
	// UpgradeName is the shared upgrade plan name for mainnet and testnet
	UpgradeName = "v1.5.0"

	// internal constants
	cliffPeriod        = int64(15552000) // 6 months in seconds
	unlockPeriod       = int64(2592000)  // 1 month in seconds
	historyStateHeight = 100000
	exp                = uint64(10e17)
	threshold          = 10 // 10 ISLM for tests
	vestingContract    = "0x40a3e24b85D32f3f68Ee9e126B8dD9dBC2D301Eb"
)

// getIgnoreList returns a static predefined list of addresses that should be ignored by the revesting upgrade
func getIgnoreList() map[string]bool {
	return map[string]bool{
		"haqq196srgtdaqrhqehdx36hfacrwmhlfznwpt78rct": true, // Team account
		"haqq1gz37yju96vhn768wncfxhrwem0pdxq0ty9v2p5": true, // Vesting Contract
	}
}

// getWhitelistedValidators returns a static predefined list of approved validators that will be bonded during the upgrade
func getWhitelistedValidators() []string {
	return []string{
		"haqqblahblahblah",
	}
}
