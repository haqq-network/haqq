package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

func (suite *KeeperTestSuite) TestParams() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	suite.Require().Equal(ethiqtypes.DefaultParams(), s.network.App.EthiqKeeper.GetParams(ctx))
	suite.Require().True(s.network.App.EthiqKeeper.IsModuleEnabled(ctx))

	np := ethiqtypes.Params{
		Enabled:      false,
		MinMintPerTx: sdkmath.OneInt().MulRaw(1e10),
		MaxMintPerTx: sdkmath.OneInt().MulRaw(1e18).MulRaw(1e5),
		MaxSupply:    sdkmath.OneInt().MulRaw(1e18).MulRaw(1e5),
	}

	suite.Require().NoError(s.network.App.EthiqKeeper.SetParams(ctx, np))
	suite.Require().False(s.network.App.EthiqKeeper.IsModuleEnabled(ctx))
	suite.Require().Equal(np, s.network.App.EthiqKeeper.GetParams(ctx))
}

// M3: Test that negative/zero parameters are rejected
func (suite *KeeperTestSuite) TestSetParams_RejectsNegativeValues() {
	suite.SetupTest()
	ctx := s.network.GetContext()
	original := s.network.App.EthiqKeeper.GetParams(ctx)

	testCases := []struct {
		name   string
		params ethiqtypes.Params
	}{
		{
			name: "negative MinMintPerTx",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(-1),
				MaxMintPerTx: sdkmath.NewInt(100),
				MaxSupply:    sdkmath.NewInt(100),
			},
		},
		{
			name: "zero MinMintPerTx",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.ZeroInt(),
				MaxMintPerTx: sdkmath.NewInt(100),
				MaxSupply:    sdkmath.NewInt(100),
			},
		},
		{
			name: "negative MaxMintPerTx",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.NewInt(-50),
				MaxSupply:    sdkmath.NewInt(100),
			},
		},
		{
			name: "zero MaxMintPerTx",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.ZeroInt(),
				MaxSupply:    sdkmath.NewInt(100),
			},
		},
		{
			name: "negative MaxSupply",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.NewInt(100),
				MaxSupply:    sdkmath.NewInt(-999),
			},
		},
		{
			name: "zero MaxSupply",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.NewInt(100),
				MaxSupply:    sdkmath.ZeroInt(),
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := s.network.App.EthiqKeeper.SetParams(ctx, tc.params)
			suite.Require().Error(err, "SetParams should reject %s", tc.name)
			suite.Require().Contains(err.Error(), "must be positive")
			// Verify params unchanged
			suite.Require().Equal(original, s.network.App.EthiqKeeper.GetParams(ctx))
		})
	}
}

// M5: Test that SetParams returns error (not void) and validates params
func (suite *KeeperTestSuite) TestSetParams_ReturnsError() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	// Valid params - should not error
	validParams := ethiqtypes.Params{
		Enabled:      true,
		MinMintPerTx: sdkmath.NewInt(1),
		MaxMintPerTx: sdkmath.NewInt(1000),
		MaxSupply:    sdkmath.NewInt(10000),
	}
	err := s.network.App.EthiqKeeper.SetParams(ctx, validParams)
	suite.Require().NoError(err, "SetParams should accept valid params")

	// Invalid params - should return error
	invalidParams := ethiqtypes.Params{
		Enabled:      true,
		MinMintPerTx: sdkmath.NewInt(-1), // Invalid
		MaxMintPerTx: sdkmath.NewInt(1000),
		MaxSupply:    sdkmath.NewInt(10000),
	}
	err = s.network.App.EthiqKeeper.SetParams(ctx, invalidParams)
	suite.Require().Error(err, "SetParams should return error for invalid params")
	suite.Require().NotNil(err)
}

// TestSetParams_UpdateScenario walks the keeper-level update path: a rejected update must
// leave the stored params untouched. The governance path is covered separately by
// TestParamsSubspaceUpdateEnforcesCrossFieldRules, which goes through the params subspace.
func (suite *KeeperTestSuite) TestSetParams_UpdateScenario() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	// Scenario 1: Valid governance proposal
	newParams := ethiqtypes.Params{
		Enabled:      true,
		MinMintPerTx: sdkmath.NewInt(100),
		MaxMintPerTx: sdkmath.NewInt(5000),
		MaxSupply:    sdkmath.NewInt(100000),
	}
	err := s.network.App.EthiqKeeper.SetParams(ctx, newParams)
	suite.Require().NoError(err)
	suite.Require().Equal(newParams, s.network.App.EthiqKeeper.GetParams(ctx))

	// Scenario 2: Invalid governance proposal (negative MaxMintPerTx)
	// This should be caught during proposal validation
	invalidParams := ethiqtypes.Params{
		Enabled:      true,
		MinMintPerTx: sdkmath.NewInt(100),
		MaxMintPerTx: sdkmath.NewInt(-5000), // Invalid!
		MaxSupply:    sdkmath.NewInt(100000),
	}
	err = s.network.App.EthiqKeeper.SetParams(ctx, invalidParams)
	suite.Require().Error(err, "Invalid params should be rejected")

	// Verify params not changed by failed update
	oldParams := newParams
	suite.Require().Equal(oldParams, s.network.App.EthiqKeeper.GetParams(ctx))

	// Scenario 3: Disable module via governance
	disabledParams := ethiqtypes.Params{
		Enabled:      false,
		MinMintPerTx: oldParams.MinMintPerTx,
		MaxMintPerTx: oldParams.MaxMintPerTx,
		MaxSupply:    oldParams.MaxSupply,
	}
	err = s.network.App.EthiqKeeper.SetParams(ctx, disabledParams)
	suite.Require().NoError(err)
	suite.Require().False(s.network.App.EthiqKeeper.IsModuleEnabled(ctx))
}

func (suite *KeeperTestSuite) TestSetParamsRejectsInvalidParams() {
	suite.SetupTest()
	ctx := s.network.GetContext()
	original := s.network.App.EthiqKeeper.GetParams(ctx)

	testCases := []struct {
		name   string
		params ethiqtypes.Params
	}{
		{
			name: "minimum equals maximum per transaction",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(10),
				MaxMintPerTx: sdkmath.NewInt(10),
				MaxSupply:    sdkmath.NewInt(100),
			},
		},
		{
			name: "maximum per transaction exceeds maximum supply",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.NewInt(101),
				MaxSupply:    sdkmath.NewInt(100),
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.Require().Error(s.network.App.EthiqKeeper.SetParams(ctx, tc.params))
			suite.Require().Equal(original, s.network.App.EthiqKeeper.GetParams(ctx))
		})
	}
}

// TestParamsSubspaceUpdateEnforcesCrossFieldRules covers the path a governance
// ParameterChangeProposal actually takes.
//
// x/params handleParameterChangeProposal calls Subspace.Update once per change, and Update runs
// only the validator registered for that key. With one key per field the cross-field rules in
// Params.Validate would never run for a governance update, so a proposal could set MaxMintPerTx
// above MaxSupply. The whole set lives under a single key precisely so this cannot happen.
func (suite *KeeperTestSuite) TestParamsSubspaceUpdateEnforcesCrossFieldRules() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	subspace := s.network.App.GetSubspace(ethiqtypes.ModuleName)
	original := s.network.App.EthiqKeeper.GetParams(ctx)

	// A proposal carries the value as JSON. Every field is spelled out, the way a proposal
	// author has to write it: Enabled is omitempty, so leaving it out keeps the stored value.
	update := func(params ethiqtypes.Params) error {
		value := []byte(fmt.Sprintf(
			`{"enabled":%t,"min_mint_per_tx":"%s","max_mint_per_tx":"%s","max_supply":"%s"}`,
			params.Enabled, params.MinMintPerTx, params.MaxMintPerTx, params.MaxSupply,
		))
		return subspace.Update(ctx, ethiqtypes.ParamStoreKeyParams, value)
	}

	testCases := []struct {
		name        string
		params      ethiqtypes.Params
		expErr      bool
		errContains string
	}{
		{
			name: "fail - max_mint_per_tx above max_supply",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.NewInt(101),
				MaxSupply:    sdkmath.NewInt(100),
			},
			expErr:      true,
			errContains: "must be less or equal to max_supply",
		},
		{
			name: "fail - min_mint_per_tx not below max_mint_per_tx",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(10),
				MaxMintPerTx: sdkmath.NewInt(10),
				MaxSupply:    sdkmath.NewInt(100),
			},
			expErr:      true,
			errContains: "must be less than max_mint_per_tx",
		},
		{
			name: "fail - zero max_supply",
			params: ethiqtypes.Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.NewInt(100),
				MaxSupply:    sdkmath.ZeroInt(),
			},
			expErr:      true,
			errContains: "must be positive",
		},
		{
			name: "success - consistent set is applied",
			params: ethiqtypes.Params{
				Enabled:      false,
				MinMintPerTx: sdkmath.NewInt(2),
				MaxMintPerTx: sdkmath.NewInt(2000),
				MaxSupply:    sdkmath.NewInt(20000),
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := update(tc.params)
			if tc.expErr {
				suite.Require().Error(err, "subspace update should reject %s", tc.name)
				suite.Require().ErrorContains(err, tc.errContains)
				suite.Require().Equal(original, s.network.App.EthiqKeeper.GetParams(ctx),
					"a rejected governance update must not change stored params")
				return
			}

			suite.Require().NoError(err)
			suite.Require().Equal(tc.params, s.network.App.EthiqKeeper.GetParams(ctx))
		})
	}
}
