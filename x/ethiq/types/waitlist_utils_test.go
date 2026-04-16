package types

import "testing"

func TestRegisteredApplicationsBySenderConsistency(t *testing.T) {
	expected := make(map[string][]uint64)

	for _, app := range registeredApplications {
		expected[app.FromAddress] = append(expected[app.FromAddress], app.ID)
	}

	if len(expected) != len(registeredApplicationsBySender) {
		t.Fatalf("sender map size mismatch: expected %d, got %d", len(expected), len(registeredApplicationsBySender))
	}

	for sender, expectedIDs := range expected {
		gotIDs, ok := registeredApplicationsBySender[sender]
		if !ok {
			t.Fatalf("sender %s is missing in registeredApplicationsBySender", sender)
		}

		if len(expectedIDs) != len(gotIDs) {
			t.Fatalf("sender %s application count mismatch: expected %d, got %d", sender, len(expectedIDs), len(gotIDs))
		}

		for i := range expectedIDs {
			if expectedIDs[i] != gotIDs[i] {
				t.Fatalf("sender %s application id mismatch at index %d: expected %d, got %d", sender, i, expectedIDs[i], gotIDs[i])
			}
		}

		if TotalNumberOfApplicationsBySender(sender) != uint64(len(expectedIDs)) {
			t.Fatalf(
				"sender %s total mismatch: expected %d, got %d",
				sender,
				len(expectedIDs),
				TotalNumberOfApplicationsBySender(sender),
			)
		}

		for i, expectedID := range expectedIDs {
			if expectedID >= uint64(len(registeredApplications)) {
				t.Fatalf("sender %s index %d points out of range app id %d", sender, i, expectedID)
			}
			if registeredApplications[expectedID].FromAddress != sender {
				t.Fatalf(
					"sender %s index %d points to app id %d owned by %s",
					sender,
					i,
					expectedID,
					registeredApplications[expectedID].FromAddress,
				)
			}
		}
	}
}
