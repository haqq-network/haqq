package tests

import (
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

var (
	UosmoDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-0",
		BaseDenom: "uosmo",
	}
	UosmoIbcdenom = UosmoDenomtrace.IBCDenom()

	UatomDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-1",
		BaseDenom: "uatom",
	}
	UatomIbcdenom = UatomDenomtrace.IBCDenom()

	UislmDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-0",
		BaseDenom: "aevmos",
	}
	UislmIbcdenom = UislmDenomtrace.IBCDenom()

	UatomOsmoDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-0/transfer/channel-1",
		BaseDenom: "uatom",
	}
	UatomOsmoIbcdenom = UatomOsmoDenomtrace.IBCDenom()

	AislmDenomtrace = transfertypes.DenomTrace{
		Path:      "transfer/channel-0",
		BaseDenom: "aevmos",
	}
	AislmIbcdenom = AislmDenomtrace.IBCDenom()
)
