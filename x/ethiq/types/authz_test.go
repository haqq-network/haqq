package types

import (
	"strings"
	"testing"
)

// canceledApplicationID returns the first withdrawn entry in the waitlist.
func canceledApplicationID(t *testing.T) uint64 {
	t.Helper()

	for id := uint64(0); id < TotalNumberOfApplications(); id++ {
		if IsApplicationCanceled(id) {
			return id
		}
	}

	t.Fatal("no canceled application in the waitlist")
	return 0
}

// liveApplicationIDs returns the first n executable entries in the waitlist.
func liveApplicationIDs(t *testing.T, n int) []uint64 {
	t.Helper()

	ids := make([]uint64, 0, n)
	for id := uint64(0); id < TotalNumberOfApplications() && len(ids) < n; id++ {
		if !IsApplicationCanceled(id) {
			ids = append(ids, id)
		}
	}
	if len(ids) < n {
		t.Fatalf("waitlist holds fewer than %d executable applications", n)
	}

	return ids
}

// TestMintHaqqByApplicationIDAuthorizationValidateBasic covers the gate on the Cosmos MsgGrant
// path: authz msg_server.Grant calls Grant.ValidateBasic, which calls this. Anything it lets
// through is written to state on every validator and walked by Accept on every use.
func TestMintHaqqByApplicationIDAuthorizationValidateBasic(t *testing.T) {
	canceled := canceledApplicationID(t)
	live := liveApplicationIDs(t, 3)
	total := TotalNumberOfApplications()

	// Longer than the waitlist, and every entry would also fail the per-ID checks. The error
	// has to name the length: that is what proves the bound is enforced before the loop
	// allocates a map sized by attacker-supplied input.
	tooLong := make([]uint64, total+1)
	for i := range tooLong {
		tooLong[i] = canceled
	}

	testCases := []struct {
		name        string
		list        []uint64
		expErr      bool
		errContains string
	}{
		{name: "single application", list: live[:1]},
		{name: "several applications", list: live},
		{
			name:        "empty list",
			list:        []uint64{},
			expErr:      true,
			errContains: "cannot be empty",
		},
		{
			name:        "nil list",
			list:        nil,
			expErr:      true,
			errContains: "cannot be empty",
		},
		{
			name:        "longer than the waitlist",
			list:        tooLong,
			expErr:      true,
			errContains: "more than the",
		},
		{
			name:        "duplicate entry",
			list:        []uint64{live[0], live[1], live[0]},
			expErr:      true,
			errContains: "duplicate application ID",
		},
		{
			name:        "unknown application",
			list:        []uint64{live[0], total},
			expErr:      true,
			errContains: "does not exist",
		},
		{
			name:        "canceled application",
			list:        []uint64{live[0], canceled},
			expErr:      true,
			errContains: "is canceled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authorization := &MintHaqqByApplicationIDAuthorization{ApplicationsList: tc.list}

			err := authorization.ValidateBasic()
			if !tc.expErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Fatalf("expected error containing %q, got: %v", tc.errContains, err)
			}
		})
	}
}

// TestIsApplicationCanceled pins the helper against the flag AsBurnApplication derives, so the
// two cannot drift apart.
func TestIsApplicationCanceled(t *testing.T) {
	for id := uint64(0); id < TotalNumberOfApplications(); id++ {
		application, err := GetApplicationByID(id)
		if err != nil {
			t.Fatalf("application %d: %v", id, err)
		}
		if got := IsApplicationCanceled(id); got != application.IsCanceled {
			t.Fatalf("application %d: IsApplicationCanceled = %v, BurnApplication.IsCanceled = %v",
				id, got, application.IsCanceled)
		}
	}

	if IsApplicationCanceled(TotalNumberOfApplications()) {
		t.Fatal("a non-existent application must not report as canceled")
	}
}
