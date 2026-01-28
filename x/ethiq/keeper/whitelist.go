package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

// registeredApplications represents the list of applications present in smart-contract.
//
// NOTE: Now here's only values for testing purposes. Final list will be set up on release.
var registeredApplications = []types.Application{
	{"haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", 1, "123000"},
	{"haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", 2, "456000"},
	{"haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", 1, "789000"},
	{"haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", 2, "1000000"},
}

// SumOfAllApplications returns sum of all applications.
func SumOfAllApplications() (sdkmath.Int, error) {
	sumAmt := sdkmath.ZeroInt()
	for i, app := range registeredApplications {
		_, _, amt, err := app.ValidateAndParse()
		if err != nil {
			return sdkmath.Int{}, errorsmod.Wrapf(err, "failed to validate and parse application entry %d", i)
		}

		sumAmt = sumAmt.Add(amt)
	}

	return sumAmt, nil
}

// SumOfAllApplicationsBeforeID returns sum of all applications submitted before given ID.
func SumOfAllApplicationsBeforeID(id uint64) (sdkmath.Int, error) {
	if uint64(len(registeredApplications)) <= id {
		return sdkmath.Int{}, errorsmod.Wrapf(types.ErrInvalidApplicationID, "application ID %d not found", id)
	}

	sumAmt := sdkmath.ZeroInt()
	for i, app := range registeredApplications {
		if uint64(i) == id {
			break
		}

		_, _, amt, err := app.ValidateAndParse()
		if err != nil {
			return sdkmath.Int{}, errorsmod.Wrapf(err, "failed to validate and parse application entry %d", i)
		}

		sumAmt = sumAmt.Add(amt)
	}

	return sumAmt, nil
}
