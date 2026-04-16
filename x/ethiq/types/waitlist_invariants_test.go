package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmdconfig "github.com/haqq-network/haqq/cmd/config"
)

func init() {
	config := sdk.GetConfig()
	cmdconfig.SetBech32Prefixes(config)
}

func TestRegisteredApplicationsInvariants(t *testing.T) {
	if len(registeredApplications) == 0 {
		t.Fatal("registeredApplications must not be empty")
	}

	if TotalNumberOfApplications() != uint64(len(registeredApplications)) {
		t.Fatalf("total applications mismatch: expected %d, got %d", len(registeredApplications), TotalNumberOfApplications())
	}

	seenIDs := make(map[uint64]struct{}, len(registeredApplications))

	for i, app := range registeredApplications {
		expectedID := uint64(i)
		if app.ID != expectedID {
			t.Fatalf("application id mismatch at index %d: expected %d, got %d", i, expectedID, app.ID)
		}

		if _, exists := seenIDs[app.ID]; exists {
			t.Fatalf("duplicate application id found: %d", app.ID)
		}
		seenIDs[app.ID] = struct{}{}

		if !IsApplicationExists(app.ID) {
			t.Fatalf("IsApplicationExists returned false for existing id %d", app.ID)
		}

		gotApp, err := GetApplicationByID(app.ID)
		if err != nil {
			t.Fatalf("GetApplicationByID failed for id %d: %v", app.ID, err)
		}
		if gotApp.Id != app.ID {
			t.Fatalf("GetApplicationByID returned wrong id: expected %d, got %d", app.ID, gotApp.Id)
		}

		if _, err := app.AsBurnApplication(); err != nil {
			t.Fatalf("invalid registered application id %d: %v", app.ID, err)
		}
	}

	if IsApplicationExists(uint64(len(registeredApplications))) {
		t.Fatalf("IsApplicationExists returned true for out-of-range id %d", len(registeredApplications))
	}
	if _, err := GetApplicationByID(uint64(len(registeredApplications))); err == nil {
		t.Fatalf("expected error for out-of-range GetApplicationByID(%d)", len(registeredApplications))
	}
}

func TestRegisteredApplicationsBySenderInvariants(t *testing.T) {
	if len(registeredApplicationsBySender) == 0 {
		t.Fatal("registeredApplicationsBySender must not be empty")
	}

	expected := make(map[string][]uint64)
	for _, app := range registeredApplications {
		expected[app.FromAddress] = append(expected[app.FromAddress], app.ID)
	}

	for sender, expectedIDs := range expected {
		gotIDs, ok := registeredApplicationsBySender[sender]
		if !ok {
			t.Fatalf("sender %s is missing in registeredApplicationsBySender", sender)
		}
		if len(gotIDs) == 0 {
			t.Fatalf("sender %s has empty application list", sender)
		}
		if len(expectedIDs) != len(gotIDs) {
			t.Fatalf("sender %s application count mismatch: expected %d, got %d", sender, len(expectedIDs), len(gotIDs))
		}
		if TotalNumberOfApplicationsBySender(sender) != uint64(len(gotIDs)) {
			t.Fatalf("sender %s total mismatch: expected %d, got %d", sender, len(gotIDs), TotalNumberOfApplicationsBySender(sender))
		}

		for i := range expectedIDs {
			if expectedIDs[i] != gotIDs[i] {
				t.Fatalf("sender %s app id mismatch at index %d: expected %d, got %d", sender, i, expectedIDs[i], gotIDs[i])
			}

			appID := gotIDs[i]
			if appID >= uint64(len(registeredApplications)) {
				t.Fatalf("sender %s index %d points out of range app id %d", sender, i, appID)
			}
			if registeredApplications[appID].FromAddress != sender {
				t.Fatalf("sender %s index %d points to app id %d owned by %s", sender, i, appID, registeredApplications[appID].FromAddress)
			}
		}

		// Boundary check for sender index lookup.
		if _, err := GetSendersApplicationIDByIndex(sender, uint64(len(gotIDs))); err == nil {
			t.Fatalf("expected out-of-range error for sender %s at index %d", sender, len(gotIDs))
		}
	}

	// Unknown sender should have zero applications and fail on indexed lookup.
	unknownSender := "haqq1zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	if TotalNumberOfApplicationsBySender(unknownSender) != 0 {
		t.Fatalf("expected zero applications for unknown sender %s", unknownSender)
	}
	if _, err := GetSendersApplicationIDByIndex(unknownSender, 0); err == nil {
		t.Fatalf("expected error for unknown sender %s", unknownSender)
	}
}

func TestGetSumOfAllApplicationsInvariant(t *testing.T) {
	sum, err := GetSumOfAllApplications()
	if err != nil {
		t.Fatalf("GetSumOfAllApplications failed: %v", err)
	}

	last := registeredApplications[len(registeredApplications)-1]
	lastBurnApp, err := last.AsBurnApplication()
	if err != nil {
		t.Fatalf("failed to parse last application: %v", err)
	}

	expected := lastBurnApp.BurnAmount.Add(lastBurnApp.BurnedBeforeAmount).Amount
	if !sum.Equal(expected) {
		t.Fatalf("sum mismatch: expected %s, got %s", expected.String(), sum.String())
	}
}
