package ucdao_test

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/ucdao"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func (s *PrecompileTestSuite) TestPrecompileAddress() {
	s.SetupTest()

	expectedAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)
	s.Require().Equal(expectedAddr, s.precompile.Address(), "precompile address should match")
}

func (s *PrecompileTestSuite) TestLoadABI() {
	abi, err := ucdao.LoadABI()
	s.Require().NoError(err, "failed to load ABI")
	s.Require().NotNil(abi, "ABI should not be nil")

	// Check that expected methods exist
	expectedMethods := []string{
		"approve",
		"revoke",
		"increaseAllowance",
		"decreaseAllowance",
		"allowance",
		"fund",
		"transferOwnership",
		"transferOwnershipWithRatio",
		"transferOwnershipWithAmount",
		"balance",
		"allBalances",
		"totalBalance",
		"enabled",
	}

	for _, method := range expectedMethods {
		_, found := abi.Methods[method]
		s.Require().True(found, "method %s should exist in ABI", method)
	}

	// Check that expected events exist
	expectedEvents := []string{
		"Approval",
		"Revocation",
		"Fund",
		"TransferOwnership",
	}

	for _, event := range expectedEvents {
		_, found := abi.Events[event]
		s.Require().True(found, "event %s should exist in ABI", event)
	}
}

func (s *PrecompileTestSuite) TestIsTransaction() {
	s.SetupTest()

	testCases := []struct {
		name          string
		methodName    string
		isTransaction bool
	}{
		// Transactions
		{"approve is transaction", "approve", true},
		{"revoke is transaction", "revoke", true},
		{"increaseAllowance is transaction", "increaseAllowance", true},
		{"decreaseAllowance is transaction", "decreaseAllowance", true},
		{"fund is transaction", "fund", true},
		{"transferOwnership is transaction", "transferOwnership", true},
		{"transferOwnershipWithRatio is transaction", "transferOwnershipWithRatio", true},
		{"transferOwnershipWithAmount is transaction", "transferOwnershipWithAmount", true},
		// Queries
		{"allowance is query", "allowance", false},
		{"balance is query", "balance", false},
		{"allBalances is query", "allBalances", false},
		{"totalBalance is query", "totalBalance", false},
		{"enabled is query", "enabled", false},
		// Unknown
		{"unknown is query", "unknownMethod", false},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := s.precompile.IsTransaction(tc.methodName)
			s.Require().Equal(tc.isTransaction, result)
		})
	}
}

func (s *PrecompileTestSuite) TestRequiredGas() {
	s.SetupTest()

	testCases := []struct {
		name     string
		input    []byte
		expected uint64
	}{
		{
			name:     "input too short",
			input:    []byte{0x01, 0x02, 0x03},
			expected: 0,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			gas := s.precompile.RequiredGas(tc.input)
			s.Require().Equal(tc.expected, gas)
		})
	}

	// Test with a valid method ID - should return non-zero gas
	abi, err := ucdao.LoadABI()
	s.Require().NoError(err)

	// Get the method ID for "enabled" (a simple query)
	enabledMethod := abi.Methods["enabled"]
	methodID := enabledMethod.ID

	gas := s.precompile.RequiredGas(methodID)
	s.Require().Greater(gas, uint64(0), "valid method should require non-zero gas")
}
