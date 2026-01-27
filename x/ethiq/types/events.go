package types

const (
	EventTypeMintExecuted                = "mint_executed"
	EventTypeMintByApplicationIDExecuted = "mint_by_application_id_executed"

	AttributeKeyHaqqMinted             = "minted"
	AttributeKeyHaqqSupplyBefore       = "supply_before"
	AttributeKeyHaqqSupplyAfter        = "supply_after"
	AttributeKeyIslmSpent              = "spent"
	AttributeKeyIslmVestingUsed        = "vesting_used"
	AttributeKeyIslmFreeUsed           = "free_used"
	AttributeKeyReceiver               = "receiver"
	AttributeKeySender                 = "sender"
	AttributeKeyApplicationID          = "application_id"
	AttributeKeyApplicationFundsSource = "application_funds_source"
)
