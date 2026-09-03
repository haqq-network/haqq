package types

import (
	"fmt"
	"testing"
)

// PushRegisteredApplicationForIntegrationTest appends one waitlist entry and returns its application ID
// and a cleanup that restores registeredApplications and registeredApplicationsBySender to the prior state.
//
// It mutates package-level variables: not safe for parallel tests or production use.
// It may only run inside binaries built with "go test" (see testing.Testing). Production binaries panic if called.
func PushRegisteredApplicationForIntegrationTest(item ApplicationListItem) (appID uint64, cleanup func()) {
	if !testing.Testing() {
		panic("types.PushRegisteredApplicationForIntegrationTest: only callable from test binaries (go test)")
	}

	if _, err := item.AsBurnApplication(); err != nil {
		panic(fmt.Sprintf("types.PushRegisteredApplicationForIntegrationTest: invalid item: %v", err))
	}

	oldApps := append([]ApplicationListItem(nil), registeredApplications...)
	oldBySender := make(map[string][]uint64, len(registeredApplicationsBySender))
	for k, v := range registeredApplicationsBySender {
		oldBySender[k] = append([]uint64(nil), v...)
	}

	appID = uint64(len(registeredApplications))
	item.ID = appID
	registeredApplications = append(registeredApplications, item)
	registeredApplicationsBySender[item.FromAddress] = append(registeredApplicationsBySender[item.FromAddress], appID)

	return appID, func() {
		registeredApplications = oldApps
		registeredApplicationsBySender = oldBySender
	}
}

func rebuildRegisteredApplicationsBySender(apps []ApplicationListItem) {
	registeredApplicationsBySender = make(map[string][]uint64, len(apps))
	for i, app := range apps {
		id := uint64(i)
		registeredApplicationsBySender[app.FromAddress] = append(registeredApplicationsBySender[app.FromAddress], id)
	}
}

// ReplaceWaitlistForIntegrationTest replaces the full waitlist and rebuilds registeredApplicationsBySender.
// Entry IDs are forced to match slice indices (0..len-1). Only for tests; not safe in parallel.
func ReplaceWaitlistForIntegrationTest(apps []ApplicationListItem) (cleanup func()) {
	if !testing.Testing() {
		panic("types.ReplaceWaitlistForIntegrationTest: only callable from test binaries (go test)")
	}

	newApps := make([]ApplicationListItem, len(apps))
	copy(newApps, apps)
	for i := range newApps {
		newApps[i].ID = uint64(i)
		if _, err := newApps[i].AsBurnApplication(); err != nil {
			panic(fmt.Sprintf("types.ReplaceWaitlistForIntegrationTest: invalid item at index %d: %v", i, err))
		}
	}

	oldApps := append([]ApplicationListItem(nil), registeredApplications...)
	oldBySender := make(map[string][]uint64, len(registeredApplicationsBySender))
	for k, v := range registeredApplicationsBySender {
		oldBySender[k] = append([]uint64(nil), v...)
	}

	registeredApplications = newApps
	rebuildRegisteredApplicationsBySender(registeredApplications)

	return func() {
		registeredApplications = oldApps
		registeredApplicationsBySender = oldBySender
	}
}
