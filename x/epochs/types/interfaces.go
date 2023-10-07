package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// EpochHooks event hooks for epoch processing
type EpochHooks interface {
	// AfterEpochEnd the first block whose timestamp is after the duration is counted as the end of the epoch
	AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64)
	// BeforeEpochStart new epoch is next block of epoch end block
	BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64)
}
