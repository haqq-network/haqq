// This file contains unit tests for the e2e package.
package upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckUpgradeProposalVersion(t *testing.T) {
	testCases := []struct {
		Name string
		Ver  string
		Exp  ProposalVersion
	}{
		{
			Name: "legacy proposal v0.47 - v1.7.1",
			Ver:  "v1.7.1",
			Exp:  LegacyProposalPreV50,
		},
		{
			Name: "normal proposal v0.46 - v1.6.4",
			Ver:  "v1.6.4",
			Exp:  LegacyProposalPreV46,
		},
		{
			Name: "normal proposal - version with whitespace - v1.6.4",
			Ver:  "\tv1.6.4 ",
			Exp:  LegacyProposalPreV46,
		},
		{
			Name: "normal proposal - version without v - 1.6.4",
			Ver:  "1.6.4",
			Exp:  LegacyProposalPreV46,
		},
		{
			Name: "SDK v0.50 proposal - version with whitespace - v1.9.0",
			Ver:  "\tv1.9.0 ",
			Exp:  UpgradeProposalV50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			legacyProposal := CheckUpgradeProposalVersion(tc.Ver)
			require.Equal(t, tc.Exp, legacyProposal, "expected: %v, got: %v", tc.Exp, legacyProposal)
		})
	}
}

// TestHaqqVersionsLess tests the HaqqVersions type's Less method with
// different version strings
func TestHaqqVersionsLess(t *testing.T) {
	var version HaqqVersions

	testCases := []struct {
		Name string
		Ver  string
		Exp  bool
	}{
		{
			Name: "higher - v1.9.0",
			Ver:  "v1.9.0",
			Exp:  false,
		},
		{
			Name: "lower - v1.7.5",
			Ver:  "v1.7.5",
			Exp:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			version = []string{tc.Ver, "v1.8.0"}
			require.Equal(t, version.Less(0, 1), tc.Exp, "expected: %v, got: %v", tc.Exp, version)
		})
	}
}

// TestHaqqVersionsSwap tests the HaqqVersions type's Swap method
func TestHaqqVersionsSwap(t *testing.T) {
	var version HaqqVersions
	value := "v1.6.4"
	version = []string{value, "v1.7.0"}
	version.Swap(0, 1)
	require.Equal(t, value, version[1], "expected: %v, got: %v", value, version[1])
}

// TestHaqqVersionsLen tests the HaqqVersions type's Len method
func TestHaqqVersionsLen(t *testing.T) {
	var version HaqqVersions = []string{"v1.6.4", "v1.7.0"}
	require.Equal(t, 2, version.Len(), "expected: %v, got: %v", 2, version.Len())
}

// TestRetrieveUpgradesList tests if the list of available upgrades in the codebase
// can be correctly retrieved
func TestRetrieveUpgradesList(t *testing.T) {
	upgradeList, err := RetrieveUpgradesList("../../../app/upgrades")
	require.NoError(t, err, "expected no error while retrieving upgrade list")
	require.NotEmpty(t, upgradeList, "expected upgrade list to be non-empty")

	// check if all entries in the list match a semantic versioning pattern
	for _, upgrade := range upgradeList {
		require.Regexp(t, `^v\d+\.\d+\.\d+(-rc\d+)*$`, upgrade, "expected upgrade version to be in semantic versioning format")
	}
}
