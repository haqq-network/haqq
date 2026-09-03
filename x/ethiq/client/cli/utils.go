package cli

import (
	"fmt"
	"strconv"
)

// parseApplicationID parses an application ID command argument.
//
// ApplicationId is uint64. Parsing through sdkmath.Int and calling Uint64 on the result takes
// the command down with a Go stack trace for anything outside that range - a negative number,
// or a value at or above 2^64 - because sdkmath.Int.Uint64 panics rather than returning an
// error. strconv.ParseUint rejects all three cases (non-numeric, negative, out of range) with
// an error the command can report.
func parseApplicationID(arg string) (uint64, error) {
	appID, err := strconv.ParseUint(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid app_id %q: expected a whole number that fits in uint64", arg)
	}

	return appID, nil
}
