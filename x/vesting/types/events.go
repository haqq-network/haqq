package types

// vesting events
const (
	EventTypeCreateClawbackVestingAccount = "create_clawback_vesting_account"
	EventTypeClawback                     = "clawback"
	EventTypeUpdateVestingFunder          = "update_vesting_funder"
	EventTypeUpdateVestingSchedule        = "update_vesting_schedule"

	AttributeKeyCoins       = "coins"
	AttributeKeyStartTime   = "start_time"
	AttributeKeyMerge       = "merge"
	AttributeKeyAccount     = "account"
	AttributeKeyFunder      = "funder"
	AttributeKeyNewFunder   = "new_funder"
	AttributeKeyDestination = "destination"
)
