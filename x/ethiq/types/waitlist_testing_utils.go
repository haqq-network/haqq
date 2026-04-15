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
