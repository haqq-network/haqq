package cli

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/x/erc20/types"
)

func TestParseMetadata(t *testing.T) {
	testCases := []struct {
		name         string
		metadataFile string
		expAmtCoins  int
		expPass      bool
	}{
		{
			"fail - invalid file name",
			"",
			0,
			false,
		},
		{
			"fail - invalid metadata",
			"metadata/invalid_metadata_test.json",
			0,
			false,
		},
		{
			"single coin metadata",
			"metadata/coin_metadata_test.json",
			1,
			true,
		},
		{
			"multiple coins metadata",
			"metadata/coins_metadata_test.json",
			2,
			true,
		},
	}
	for _, tc := range testCases {
		metadata, err := ParseMetadata(types.AminoCdc, tc.metadataFile)
		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, tc.expAmtCoins, len(metadata))
		} else {
			require.Error(t, err)
		}
	}
}
