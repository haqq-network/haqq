package cli

import (
	"strings"
	"testing"
)

// TestParseApplicationID covers the argument forms that used to take the command down.
//
// The previous parse went through sdkmath.Int and called Uint64 on the result, which panics
// outside the uint64 range instead of returning an error: `mint-by-app 18446744073709551616`
// answered with a Go stack trace.
func TestParseApplicationID(t *testing.T) {
	testCases := []struct {
		name   string
		arg    string
		expID  uint64
		expErr bool
	}{
		{name: "zero", arg: "0", expID: 0},
		{name: "small id", arg: "42", expID: 42},
		{name: "max uint64", arg: "18446744073709551615", expID: ^uint64(0)},
		{name: "one above max uint64", arg: "18446744073709551616", expErr: true},
		{name: "far above max uint64", arg: "18446744073709551621", expErr: true},
		{name: "negative", arg: "-1", expErr: true},
		{name: "empty", arg: "", expErr: true},
		{name: "not a number", arg: "abc", expErr: true},
		{name: "decimal", arg: "1.5", expErr: true},
		{name: "leading plus", arg: "+1", expErr: true},
		{name: "whitespace", arg: " 1", expErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("parseApplicationID panicked instead of returning an error: %v", r)
				}
			}()

			appID, err := parseApplicationID(tc.arg)
			if tc.expErr {
				if err == nil {
					t.Fatalf("expected an error for %q", tc.arg)
				}
				if !strings.Contains(err.Error(), "invalid app_id") {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if appID != tc.expID {
				t.Fatalf("expected %d, got %d", tc.expID, appID)
			}
		})
	}
}
