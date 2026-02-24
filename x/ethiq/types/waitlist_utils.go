package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
)

var FundSources = map[SourceOfFunds]string{
	SourceOfFunds_SOURCE_OF_FUNDS_BANK:  "BANK",
	SourceOfFunds_SOURCE_OF_FUNDS_UCDAO: "UCDAO",
}

func IsApplicationExists(appID uint64) bool {
	return appID < uint64(len(registeredApplications))
}

func TotalNumberOfApplications() int {
	return len(registeredApplications)
}

func TotalNumberOfApplicationsBySender(sender string) int {
	apps, ok := registeredApplicationsBySender[sender]
	if !ok {
		return 0
	}

	return len(apps)
}

func GetLastApplication() ApplicationListItem {
	return registeredApplications[len(registeredApplications)-1]
}

func GetApplicationByID(appID uint64) (*BurnApplication, error) {
	if !IsApplicationExists(appID) {
		return nil, errorsmod.Wrapf(ErrInvalidApplicationID, "application ID %d not found", appID)
	}

	appItem := registeredApplications[appID]

	return appItem.AsBurnApplication()
}

func GetSendersApplicationIDByIndex(sender string, i int) (*BurnApplication, error) {
	apps, ok := registeredApplicationsBySender[sender]
	if !ok {
		return nil, errorsmod.Wrapf(ErrInvalidApplicationID, "applications for %s not found", sender)
	}

	if i >= len(apps) {
		return nil, errorsmod.Wrapf(ErrInvalidApplicationID, "out of range, %s has %d applications, requested %d", sender, len(apps), i)
	}

	appID := apps[i]

	return GetApplicationByID(appID)
}

func GetSumOfAllApplications() sdkmath.Int {
	lastApplication := GetLastApplication()
	burnApplication, err := lastApplication.AsBurnApplication()
	if err != nil {
		// should never happen
		panic(err)
	}

	sum := burnApplication.BurnAmount.Add(burnApplication.BurnedBeforeAmount)

	return sum.Amount
}
