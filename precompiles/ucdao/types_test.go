package ucdao_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/precompiles/ucdao"
)

func TestParseApproveArgs(t *testing.T) {
	validSpender := common.HexToAddress("0x1234567890123456789012345678901234567890")
	validCoins := []struct {
		Denom  string   `json:"denom"`
		Amount *big.Int `json:"amount"`
	}{
		{Denom: "aISLM", Amount: big.NewInt(1000)},
	}

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name: "pass - valid args",
			args: []interface{}{
				validSpender,
				validCoins,
			},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validSpender},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name: "fail - empty spender address",
			args: []interface{}{
				common.Address{},
				validCoins,
			},
			expPass:     false,
			errContains: "invalid spender address",
		},
		{
			name: "fail - invalid coins type",
			args: []interface{}{
				validSpender,
				"not a coin array",
			},
			expPass:     false,
			errContains: "invalid coins argument",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spender, coins, err := ucdao.ParseApproveArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, validSpender, spender)
				require.Equal(t, 1, len(coins))
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseRevokeArgs(t *testing.T) {
	validSpender := common.HexToAddress("0x1234567890123456789012345678901234567890")

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name:    "pass - valid args",
			args:    []interface{}{validSpender},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validSpender, "extra"},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name:        "fail - empty spender address",
			args:        []interface{}{common.Address{}},
			expPass:     false,
			errContains: "invalid spender address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spender, err := ucdao.ParseRevokeArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, validSpender, spender)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseAllowanceArgs(t *testing.T) {
	validOwner := common.HexToAddress("0x1234567890123456789012345678901234567890")
	validSpender := common.HexToAddress("0x0987654321098765432109876543210987654321")

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name:    "pass - valid args",
			args:    []interface{}{validOwner, validSpender},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validOwner},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name:        "fail - empty owner address",
			args:        []interface{}{common.Address{}, validSpender},
			expPass:     false,
			errContains: "invalid owner address",
		},
		{
			name:        "fail - empty spender address",
			args:        []interface{}{validOwner, common.Address{}},
			expPass:     false,
			errContains: "invalid spender address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner, spender, err := ucdao.ParseAllowanceArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, validOwner, owner)
				require.Equal(t, validSpender, spender)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseFundArgs(t *testing.T) {
	validCoins := []struct {
		Denom  string   `json:"denom"`
		Amount *big.Int `json:"amount"`
	}{
		{Denom: "aISLM", Amount: big.NewInt(1000)},
	}

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name:    "pass - valid args",
			args:    []interface{}{validCoins},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validCoins, "extra"},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name:        "fail - invalid coins type",
			args:        []interface{}{"not a coin array"},
			expPass:     false,
			errContains: "invalid coins argument",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			coins, err := ucdao.ParseFundArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, 1, len(coins))
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseTransferOwnershipArgs(t *testing.T) {
	validOwner := common.HexToAddress("0x1234567890123456789012345678901234567890")
	validNewOwner := common.HexToAddress("0x0987654321098765432109876543210987654321")

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name:    "pass - valid args",
			args:    []interface{}{validOwner, validNewOwner},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validOwner},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name:        "fail - empty owner address",
			args:        []interface{}{common.Address{}, validNewOwner},
			expPass:     false,
			errContains: "invalid owner address",
		},
		{
			name:        "fail - empty new owner address",
			args:        []interface{}{validOwner, common.Address{}},
			expPass:     false,
			errContains: "invalid new owner address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner, newOwner, err := ucdao.ParseTransferOwnershipArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, validOwner, owner)
				require.Equal(t, validNewOwner, newOwner)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseTransferOwnershipWithRatioArgs(t *testing.T) {
	validOwner := common.HexToAddress("0x1234567890123456789012345678901234567890")
	validNewOwner := common.HexToAddress("0x0987654321098765432109876543210987654321")
	validRatio := big.NewInt(500000000000000000) // 0.5 in 1e18 precision

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name:    "pass - valid args",
			args:    []interface{}{validOwner, validNewOwner, validRatio},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validOwner, validNewOwner},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name:        "fail - empty owner address",
			args:        []interface{}{common.Address{}, validNewOwner, validRatio},
			expPass:     false,
			errContains: "invalid owner address",
		},
		{
			name:        "fail - invalid ratio type",
			args:        []interface{}{validOwner, validNewOwner, "not a big.Int"},
			expPass:     false,
			errContains: "invalid ratio",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner, newOwner, ratio, err := ucdao.ParseTransferOwnershipWithRatioArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, validOwner, owner)
				require.Equal(t, validNewOwner, newOwner)
				require.False(t, ratio.IsNil())
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseTransferOwnershipWithAmountArgs(t *testing.T) {
	validOwner := common.HexToAddress("0x1234567890123456789012345678901234567890")
	validNewOwner := common.HexToAddress("0x0987654321098765432109876543210987654321")
	validCoins := []struct {
		Denom  string   `json:"denom"`
		Amount *big.Int `json:"amount"`
	}{
		{Denom: "aISLM", Amount: big.NewInt(1000)},
	}

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name:    "pass - valid args",
			args:    []interface{}{validOwner, validNewOwner, validCoins},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validOwner, validNewOwner},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name:        "fail - empty owner address",
			args:        []interface{}{common.Address{}, validNewOwner, validCoins},
			expPass:     false,
			errContains: "invalid owner address",
		},
		{
			name:        "fail - invalid coins type",
			args:        []interface{}{validOwner, validNewOwner, "not coins"},
			expPass:     false,
			errContains: "invalid coins argument",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner, newOwner, coins, err := ucdao.ParseTransferOwnershipWithAmountArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, validOwner, owner)
				require.Equal(t, validNewOwner, newOwner)
				require.Equal(t, 1, len(coins))
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseBalanceArgs(t *testing.T) {
	validAccount := common.HexToAddress("0x1234567890123456789012345678901234567890")

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name:    "pass - valid args",
			args:    []interface{}{validAccount, "aISLM"},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validAccount},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name:        "fail - empty account address",
			args:        []interface{}{common.Address{}, "aISLM"},
			expPass:     false,
			errContains: "invalid account address",
		},
		{
			name:        "fail - invalid denom type",
			args:        []interface{}{validAccount, 123},
			expPass:     false,
			errContains: "invalid denom",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			account, denom, err := ucdao.ParseBalanceArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, validAccount, account)
				require.Equal(t, "aISLM", denom)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseAllBalancesArgs(t *testing.T) {
	validAccount := common.HexToAddress("0x1234567890123456789012345678901234567890")

	testCases := []struct {
		name        string
		args        []interface{}
		expPass     bool
		errContains string
	}{
		{
			name:    "pass - valid args",
			args:    []interface{}{validAccount},
			expPass: true,
		},
		{
			name:        "fail - invalid number of args",
			args:        []interface{}{validAccount, "extra"},
			expPass:     false,
			errContains: "invalid number of arguments",
		},
		{
			name:        "fail - empty account address",
			args:        []interface{}{common.Address{}},
			expPass:     false,
			errContains: "invalid account address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			account, err := ucdao.ParseAllBalancesArgs(tc.args)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, validAccount, account)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

func TestParseCoinsArg(t *testing.T) {
	testCases := []struct {
		name        string
		arg         interface{}
		expPass     bool
		expLen      int
		errContains string
	}{
		{
			name: "pass - valid coins",
			arg: []struct {
				Denom  string   `json:"denom"`
				Amount *big.Int `json:"amount"`
			}{
				{Denom: "aISLM", Amount: big.NewInt(1000)},
				{Denom: "aLIQUID1", Amount: big.NewInt(500)},
			},
			expPass: true,
			expLen:  2,
		},
		{
			name: "pass - empty coins",
			arg: []struct {
				Denom  string   `json:"denom"`
				Amount *big.Int `json:"amount"`
			}{},
			expPass: true,
			expLen:  0,
		},
		{
			name: "fail - negative amount",
			arg: []struct {
				Denom  string   `json:"denom"`
				Amount *big.Int `json:"amount"`
			}{
				{Denom: "aISLM", Amount: big.NewInt(-1000)},
			},
			expPass:     false,
			errContains: "invalid coin amount",
		},
		{
			name: "fail - nil amount",
			arg: []struct {
				Denom  string   `json:"denom"`
				Amount *big.Int `json:"amount"`
			}{
				{Denom: "aISLM", Amount: nil},
			},
			expPass:     false,
			errContains: "invalid coin amount",
		},
		{
			name:        "fail - invalid type",
			arg:         "not a coin array",
			expPass:     false,
			errContains: "invalid coins argument",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			coins, err := ucdao.ParseCoinsArg(tc.arg)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, tc.expLen, len(coins))
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}
