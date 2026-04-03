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

func TotalNumberOfApplications() uint64 {
	return uint64(len(registeredApplications))
}

func TotalNumberOfApplicationsBySender(sender string) uint64 {
	apps, ok := registeredApplicationsBySender[sender]
	if !ok {
		return 0
	}

	return uint64(len(apps))
}

func GetLastApplication() ApplicationListItem {
	return registeredApplications[len(registeredApplications)-1]
}

func GetApplicationByID(appID uint64) (*BurnApplication, error) {
	if !IsApplicationExists(appID) {
		return nil, errorsmod.Wrapf(ErrInvalidApplicationID, "application not found: id = %d", appID)
	}

	appItem := registeredApplications[appID]

	return appItem.AsBurnApplication()
}

func GetSendersApplicationIDByIndex(sender string, i uint64) (*BurnApplication, error) {
	apps, ok := registeredApplicationsBySender[sender]
	if !ok {
		return nil, errorsmod.Wrapf(ErrInvalidApplicationID, "applications for %s not found", sender)
	}

	if i >= uint64(len(apps)) {
		return nil, errorsmod.Wrapf(ErrInvalidApplicationID, "out of range, %s has %d applications, requested %d", sender, len(apps), i)
	}

	return GetApplicationByID(apps[i])
}

func GetSumOfAllApplications() (sdkmath.Int, error) {
	lastApplication := GetLastApplication()
	burnApplication, err := lastApplication.AsBurnApplication()
	if err != nil {
		return sdkmath.Int{}, errorsmod.Wrap(err, "failed to parse last application")
	}

	sum := burnApplication.BurnAmount.Add(burnApplication.BurnedBeforeAmount)

	return sum.Amount, nil
}
