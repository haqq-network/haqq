package tests

import (
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

var (
	UosmoDenomtrace = transfertypes.ExtractDenomFromPath("transfer/channel-0/uosmo")
	UosmoIbcdenom   = UosmoDenomtrace.IBCDenom()

	UatomDenomtrace = transfertypes.ExtractDenomFromPath("transfer/channel-1/uatom")
	UatomIbcdenom   = UatomDenomtrace.IBCDenom()

	UislmDenomtrace = transfertypes.ExtractDenomFromPath("transfer/channel-0/aISLM")
	UislmIbcdenom   = UislmDenomtrace.IBCDenom()

	UatomOsmoDenomtrace = transfertypes.ExtractDenomFromPath("transfer/channel-0/transfer/channel-1/uatom")
	UatomOsmoIbcdenom   = UatomOsmoDenomtrace.IBCDenom()

	AislmDenomtrace = transfertypes.ExtractDenomFromPath("transfer/channel-0/aISLM")
	AislmIbcdenom   = AislmDenomtrace.IBCDenom()
)
