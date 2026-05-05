package liquid_test

import (
	"fmt"
	"math/big"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	sdkauthz "github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/liquid"
	"github.com/haqq-network/haqq/precompiles/testutil"
	haqqtestutil "github.com/haqq-network/haqq/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/utils"
	liquidtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// vestingAmount is the total amount used for vesting account tests.
var vestingAmount = sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 3_000_000))

// createClawbackVestingAccount creates a ClawbackVestingAccount for the given address
// with all tokens locked (lockup starts 10 seconds ago so the account is mid-lockup).
func (s *PrecompileTestSuite) createClawbackVestingAccount(ctx sdk.Context, addr sdk.AccAddress) {
	funder := sdk.AccAddress(liquidtypes.ModuleName)
	lockupPeriods := sdkvesting.Periods{
		{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
		{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
		{Length: 100000, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
	}
	vestingPeriods := sdkvesting.Periods{
		{Length: 0, Amount: vestingAmount},
	}

	baseAccount := authtypes.NewBaseAccountWithAddress(addr)
	baseAccount.AccountNumber = s.network.App.AccountKeeper.NextAccountNumber(ctx)
	startTime := ctx.BlockTime().Add(-10 * time.Second)
	clawbackAccount := vestingtypes.NewClawbackVestingAccount(
		baseAccount, funder, vestingAmount, startTime, lockupPeriods, vestingPeriods, nil,
	)
	err := haqqtestutil.FundAccount(ctx, s.network.App.BankKeeper, addr, vestingAmount)
	s.Require().NoError(err)
	s.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
}

// smartContractVestingTotal is the total amount used by the smart-contract
// vesting test factory: 3000 ISLM = 3000 * 1e18 aISLM.
var smartContractVestingTotal = sdk.NewCoins(
	sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(3000).MulRaw(1e18)),
)

// createClawbackVestingAccountForSmartContract converts an existing on-chain
// account at addr (typically a deployed smart-contract wallet such as a Gnosis
// Safe proxy) into a ClawbackVestingAccount with the schedule used by Safe
// integration tests: 3 lockup periods of 1000 ISLM each (3000 ISLM total) and
// a single instant vesting period for 3000 ISLM. The lockup starts 10 seconds
// in the past, so all 3000 ISLM are still locked at block time.
//
// IMPORTANT - test-only helper.
//
// In production this conversion is impossible: x/vesting MsgConvertIntoVestingAccount
// rejects contract accounts (see the IsContractAccount guard in
// x/vesting/keeper/msg_server.go). The helper deliberately bypasses that
// guard by mutating the auth account directly. It MUST NOT be treated as a
// model for any on-chain flow - it only exists to build a test fixture
// where a smart-contract wallet behaves as if it were under a vesting
// schedule, so the liquid vesting precompile can be exercised end-to-end
// against a Safe-shaped sender.
//
// Side effects:
//   - mints 3000 ISLM into addr via testutil.FundAccount (on top of any
//     pre-existing balance);
//   - replaces the auth account at addr with a ClawbackVestingAccount that
//     preserves the original BaseAccount metadata (AccountNumber, Sequence,
//     PubKey) and the original CodeHash if addr was an EthAccount, so the
//     contract code at that address remains callable through the EVM.
func (s *PrecompileTestSuite) createClawbackVestingAccountForSmartContract(
	ctx sdk.Context, addr sdk.AccAddress,
) {
	funder := sdk.AccAddress(liquidtypes.ModuleName)
	periodCoin := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000).MulRaw(1e18)))
	lockupPeriods := sdkvesting.Periods{
		{Length: 100000, Amount: periodCoin},
		{Length: 100000, Amount: periodCoin},
		{Length: 100000, Amount: periodCoin},
	}
	vestingPeriods := sdkvesting.Periods{
		{Length: 0, Amount: smartContractVestingTotal},
	}

	existing := s.network.App.AccountKeeper.GetAccount(ctx, addr)
	s.Require().NotNil(existing, "smart-contract account must exist before being wrapped into a vesting account")

	baseAccount := authtypes.NewBaseAccount(
		existing.GetAddress(),
		existing.GetPubKey(),
		existing.GetAccountNumber(),
		existing.GetSequence(),
	)

	// Preserve the EVM code reference. ClawbackVestingAccount stores its own
	// CodeHash field, so we must carry it over from the EthAccount to keep
	// the contract callable via the EVM.
	var codeHashPtr *common.Hash
	if ethAcc, ok := existing.(*haqqtypes.EthAccount); ok {
		h := ethAcc.GetCodeHash()
		codeHashPtr = &h
	}

	startTime := ctx.BlockTime().Add(-10 * time.Second)
	clawbackAccount := vestingtypes.NewClawbackVestingAccount(
		baseAccount, funder, smartContractVestingTotal, startTime, lockupPeriods, vestingPeriods, codeHashPtr,
	)

	s.Require().NoError(haqqtestutil.FundAccount(ctx, s.network.App.BankKeeper, addr, smartContractVestingTotal))
	s.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
}

// ---------------------------------------------------------------------------
// TestNewLiquidateMsg covers types.go parsing.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestNewLiquidateMsg() {
	from := s.keyring.GetAddr(0)
	to := s.keyring.GetAddr(1)

	testCases := []struct {
		name        string
		args        []any
		expError    bool
		errContains string
	}{
		{
			"fail - wrong arg count",
			[]any{},
			true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 3, 0),
		},
		{
			"fail - invalid from (non-address)",
			[]any{"not-an-addr", to, big.NewInt(1_000_000)},
			true,
			"invalid sender",
		},
		{
			"fail - invalid to (non-address)",
			[]any{from, "not-an-addr", big.NewInt(1_000_000)},
			true,
			"invalid receiver",
		},
		{
			"fail - nil amount",
			[]any{from, to, nil},
			true,
			"invalid amount",
		},
		{
			"success",
			[]any{from, to, big.NewInt(1_000_000)},
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg, retFrom, retTo, err := liquid.NewLiquidateMsg(tc.args)
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().Equal(from, retFrom)
				s.Require().Equal(to, retTo)
				s.Require().NotNil(msg)
				s.Require().Equal(sdkmath.NewInt(1_000_000), msg.Amount.Amount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestNewRedeemMsg covers types.go parsing.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestNewRedeemMsg() {
	from := s.keyring.GetAddr(0)
	to := s.keyring.GetAddr(1)

	testCases := []struct {
		name        string
		args        []any
		expError    bool
		errContains string
	}{
		{
			"fail - wrong arg count",
			[]any{},
			true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 4, 0),
		},
		{
			"fail - invalid from (non-address)",
			[]any{"not-an-addr", to, liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000)},
			true,
			"invalid sender",
		},
		{
			"fail - invalid to (non-address)",
			[]any{from, "not-an-addr", liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000)},
			true,
			"invalid receiver",
		},
		{
			"fail - invalid denom type",
			[]any{from, to, 123, big.NewInt(1_000_000)},
			true,
			"invalid denom",
		},
		{
			"fail - nil amount",
			[]any{from, to, liquidtypes.DenomBaseNameFromID(0), nil},
			true,
			"invalid amount",
		},
		{
			"success",
			[]any{from, to, liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000)},
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg, retFrom, retTo, err := liquid.NewRedeemMsg(tc.args)
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().Equal(from, retFrom)
				s.Require().Equal(to, retTo)
				s.Require().NotNil(msg)
				s.Require().Equal(sdkmath.NewInt(1_000_000), msg.Amount.Amount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestLiquidate covers the Liquidate precompile method.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestLiquidate() {
	method := s.precompile.Methods[liquid.LiquidateMethod]

	testCases := []struct {
		name        string
		malleate    func() []any
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []any { return []any{} },
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 3, 0),
		},
		{
			"fail - invalid from address type",
			func() []any {
				return []any{"not-an-addr", s.keyring.GetAddr(1), big.NewInt(1_000_000)}
			},
			200000, true,
			"invalid sender",
		},
		{
			"fail - invalid to address type",
			func() []any {
				return []any{s.keyring.GetAddr(0), "not-an-addr", big.NewInt(1_000_000)}
			},
			200000, true,
			"invalid receiver",
		},
		{
			"fail - nil amount",
			func() []any {
				return []any{s.keyring.GetAddr(0), s.keyring.GetAddr(1), nil}
			},
			200000, true,
			"invalid amount",
		},
		{
			"fail - zero amount (ValidateBasic rejects)",
			func() []any {
				return []any{s.keyring.GetAddr(0), s.keyring.GetAddr(1), big.NewInt(0)}
			},
			200000, true,
			"",
		},
		{
			"fail - different origin from sender",
			func() []any {
				differentAddr := utiltx.GenerateAddress()
				return []any{differentAddr, s.keyring.GetAddr(1), big.NewInt(1_000_000)}
			},
			200000, true,
			"origin address",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx := s.network.GetContext()
			origin := s.keyring.GetAddr(0)

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, origin, s.precompile, tc.gas)
			_, err := s.precompile.Liquidate(ctx, origin, contract, s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// TestLiquidateSuccess tests a successful liquidation end-to-end.
func (s *PrecompileTestSuite) TestLiquidateSuccess() {
	s.SetupTest()
	ctx := s.network.GetContext()

	// Create a fresh address for the vesting account (not in keyring)
	fromEvmAddr := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(fromEvmAddr.Bytes())
	toAddr := s.keyring.GetAddr(1)

	// Set up ClawbackVestingAccount for fromAddr
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	method := s.precompile.Methods[liquid.LiquidateMethod]
	// CallerAddress == sender == origin → no authz needed
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, fromEvmAddr, s.precompile, 500000)

	bz, err := s.precompile.Liquidate(ctx, fromEvmAddr, contract, s.network.GetStateDB(), &method, []any{
		fromEvmAddr,
		toAddr,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err)
	s.Require().NotNil(bz)

	// Unpack the result: (uint256 minted, address erc20Contract)
	results, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Len(results, 2)
	minted, ok := results[0].(*big.Int)
	s.Require().True(ok)
	s.Require().Equal(big.NewInt(1_000_000), minted)
}

// ---------------------------------------------------------------------------
// TestRedeem covers the Redeem precompile method.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestRedeem() {
	method := s.precompile.Methods[liquid.RedeemMethod]

	testCases := []struct {
		name        string
		malleate    func() []any
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []any { return []any{} },
			200000, true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 4, 0),
		},
		{
			"fail - invalid from address type",
			func() []any {
				return []any{"not-an-addr", s.keyring.GetAddr(1), liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000)}
			},
			200000, true,
			"invalid sender",
		},
		{
			"fail - invalid to address type",
			func() []any {
				return []any{s.keyring.GetAddr(0), "not-an-addr", liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000)}
			},
			200000, true,
			"invalid receiver",
		},
		{
			"fail - invalid denom type",
			func() []any {
				return []any{s.keyring.GetAddr(0), s.keyring.GetAddr(1), 123, big.NewInt(1_000_000)}
			},
			200000, true,
			"invalid denom",
		},
		{
			"fail - nil amount",
			func() []any {
				return []any{s.keyring.GetAddr(0), s.keyring.GetAddr(1), liquidtypes.DenomBaseNameFromID(0), nil}
			},
			200000, true,
			"invalid amount",
		},
		{
			"fail - zero amount (ValidateBasic rejects)",
			func() []any {
				return []any{s.keyring.GetAddr(0), s.keyring.GetAddr(1), liquidtypes.DenomBaseNameFromID(0), big.NewInt(0)}
			},
			200000, true,
			"",
		},
		{
			"fail - different origin from sender",
			func() []any {
				differentAddr := utiltx.GenerateAddress()
				return []any{differentAddr, s.keyring.GetAddr(1), liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000)}
			},
			200000, true,
			"origin address",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx := s.network.GetContext()
			origin := s.keyring.GetAddr(0)

			contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, origin, s.precompile, tc.gas)
			_, err := s.precompile.Redeem(ctx, origin, contract, s.network.GetStateDB(), &method, tc.malleate())
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// TestRedeemSuccess tests a successful redeem end-to-end (liquidate first, then redeem).
func (s *PrecompileTestSuite) TestRedeemSuccess() {
	s.SetupTest()
	ctx := s.network.GetContext()

	// Create vesting account and liquidate to get liquid tokens
	fromEvmAddr := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(fromEvmAddr.Bytes())
	toEvmAddr := s.keyring.GetAddr(1)
	toAccAddr := sdk.AccAddress(toEvmAddr.Bytes())

	s.createClawbackVestingAccount(ctx, fromAccAddr)

	// Step 1: Liquidate to produce liquid tokens in toAddr
	liquidateMethod := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, fromEvmAddr, s.precompile, 500000)
	bz, err := s.precompile.Liquidate(ctx, fromEvmAddr, contract, s.network.GetStateDB(), &liquidateMethod, []any{
		fromEvmAddr,
		toEvmAddr,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err)
	s.Require().NotNil(bz)

	// Verify toAddr has the liquid denom
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	balance := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, liquidDenom)
	s.Require().Equal(sdkmath.NewInt(1_000_000), balance.Amount)

	// Step 2: Redeem the liquid tokens back to native
	redeemMethod := s.precompile.Methods[liquid.RedeemMethod]
	redeemContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, toEvmAddr, s.precompile, 500000)
	redeemBz, err := s.precompile.Redeem(ctx, toEvmAddr, redeemContract, s.network.GetStateDB(), &redeemMethod, []any{
		toEvmAddr,
		toEvmAddr,
		liquidDenom,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err)
	// Redeem has no outputs, so Pack() returns nil bytes — that is expected.
	_ = redeemBz
}

// ---------------------------------------------------------------------------
// TestRequiredGas covers the RequiredGas method.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestRequiredGas() {
	// Input shorter than 4 bytes → 0
	s.Require().Equal(uint64(0), s.precompile.RequiredGas([]byte{}))
	s.Require().Equal(uint64(0), s.precompile.RequiredGas([]byte{1, 2, 3}))

	// Unknown 4-byte method ID → 0
	s.Require().Equal(uint64(0), s.precompile.RequiredGas([]byte{0xFF, 0xFF, 0xFF, 0xFF}))

	// Valid method IDs return non-zero gas
	liquidateID := s.precompile.ABI.Methods[liquid.LiquidateMethod].ID
	s.Require().Greater(s.precompile.RequiredGas(liquidateID), uint64(0))

	redeemID := s.precompile.ABI.Methods[liquid.RedeemMethod].ID
	s.Require().Greater(s.precompile.RequiredGas(redeemID), uint64(0))
}

// ---------------------------------------------------------------------------
// TestLiquidateViaContract tests Liquidate when caller != origin (authz path).
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestLiquidateViaContract() {
	s.SetupTest()
	ctx := s.network.GetContext()

	fromEvmAddr := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(fromEvmAddr.Bytes())
	toAddr := s.keyring.GetAddr(1)

	// The "contract" is a separate address acting as the caller.
	contractEvmAddr := utiltx.GenerateAddress()
	contractAccAddr := sdk.AccAddress(contractEvmAddr.Bytes())

	s.createClawbackVestingAccount(ctx, fromAccAddr)

	// Grant a GenericAuthorization from fromAddr (granter/origin) to contractAddr (grantee/caller).
	msgURL := sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{})
	expiration := time.Now().Add(time.Hour)
	grant := sdkauthz.NewGenericAuthorization(msgURL)
	err := s.network.App.AuthzKeeper.SaveGrant(ctx, contractAccAddr, fromAccAddr, grant, &expiration)
	s.Require().NoError(err)

	method := s.precompile.Methods[liquid.LiquidateMethod]
	// CallerAddress = contractEvmAddr (different from origin = fromEvmAddr)
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, contractEvmAddr, s.precompile, 500000)
	bz, err := s.precompile.Liquidate(ctx, fromEvmAddr, contract, s.network.GetStateDB(), &method, []any{
		fromEvmAddr,
		toAddr,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err)
	s.Require().NotNil(bz)
}

// ---------------------------------------------------------------------------
// TestLiquidateViaContractNoGrant verifies the "no authz grant" error path.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestLiquidateViaContractNoGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	originEvmAddr := utiltx.GenerateAddress()
	callerEvmAddr := utiltx.GenerateAddress() // different from origin → authz check triggered

	fromAccAddr := sdk.AccAddress(originEvmAddr.Bytes())
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	method := s.precompile.Methods[liquid.LiquidateMethod]
	// CallerAddress = callerEvmAddr, origin = originEvmAddr — no grant saved.
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerEvmAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, originEvmAddr, contract, s.network.GetStateDB(), &method, []any{
		originEvmAddr,
		s.keyring.GetAddr(1),
		big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "does not exist or is expired")
}

// ---------------------------------------------------------------------------
// TestRedeemViaContractNoGrant verifies the "no authz grant" error path in Redeem.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestRedeemViaContractNoGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	originEvmAddr := utiltx.GenerateAddress()
	callerEvmAddr := utiltx.GenerateAddress()

	method := s.precompile.Methods[liquid.RedeemMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerEvmAddr, s.precompile, 500000)
	_, err := s.precompile.Redeem(ctx, originEvmAddr, contract, s.network.GetStateDB(), &method, []any{
		originEvmAddr,
		originEvmAddr,
		liquidtypes.DenomBaseNameFromID(0),
		big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "does not exist or is expired")
}

// ---------------------------------------------------------------------------
// TestRedeemViaContract tests Redeem when caller != origin (authz path).
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestRedeemViaContract() {
	s.SetupTest()
	ctx := s.network.GetContext()

	fromEvmAddr := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(fromEvmAddr.Bytes())
	toEvmAddr := s.keyring.GetAddr(1)
	toAccAddr := sdk.AccAddress(toEvmAddr.Bytes())

	contractEvmAddr := utiltx.GenerateAddress()
	contractAccAddr := sdk.AccAddress(contractEvmAddr.Bytes())

	s.createClawbackVestingAccount(ctx, fromAccAddr)

	// Step 1: Liquidate directly (caller == origin) to produce liquid tokens.
	liquidateMethod := s.precompile.Methods[liquid.LiquidateMethod]
	liqContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, fromEvmAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, fromEvmAddr, liqContract, s.network.GetStateDB(), &liquidateMethod, []any{
		fromEvmAddr,
		toEvmAddr,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err)

	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	balance := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, liquidDenom)
	s.Require().Equal(sdkmath.NewInt(1_000_000), balance.Amount)

	// Step 2: Grant a GenericAuthorization for Redeem from toAddr to contractAddr.
	redeemURL := sdk.MsgTypeURL(&liquidtypes.MsgRedeem{})
	expiration := time.Now().Add(time.Hour)
	redeemGrant := sdkauthz.NewGenericAuthorization(redeemURL)
	err = s.network.App.AuthzKeeper.SaveGrant(ctx, contractAccAddr, toAccAddr, redeemGrant, &expiration)
	s.Require().NoError(err)

	// Step 3: Redeem via the contract (caller = contractEvmAddr, origin = toEvmAddr).
	redeemMethod := s.precompile.Methods[liquid.RedeemMethod]
	redeemContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, contractEvmAddr, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, toEvmAddr, redeemContract, s.network.GetStateDB(), &redeemMethod, []any{
		toEvmAddr,
		toEvmAddr,
		liquidDenom,
		big.NewInt(1_000_000),
	})
	s.Require().NoError(err)
}

// ---------------------------------------------------------------------------
// Mirror / state-consistency tests: explicit bank-delta assertions for the
// EOA-driven Liquidate / Redeem paths. These complement the Safe integration
// suite: there caller != origin and the mirror MUST fire; here caller ==
// origin and the mirror MUST be a no-op. A regression in either direction
// (mirror leaks for EOA, or mirror is silently skipped for contract) shows up
// here as a wrong post-call bank balance.
// ---------------------------------------------------------------------------

// TestLiquidateEOABankDeltas verifies that an EOA-driven Liquidate produces
// exactly the expected bank deltas on liquidateFrom (debited base denom),
// liquidateTo (credited liquid denom only) and the liquid vesting module
// (credited base denom). The mirror is a no-op on the caller==origin path,
// so any over- or under-counting would show up as a wrong bank balance.
func (s *PrecompileTestSuite) TestLiquidateEOABankDeltas() {
	s.SetupTest()
	ctx := s.network.GetContext()

	// liquidateFrom is a fresh vesting account; liquidateTo is a distinct
	// keyring EOA so we also exercise the from != to branch on the EOA path.
	fromEvmAddr := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(fromEvmAddr.Bytes())
	toEvmAddr := s.keyring.GetAddr(1)
	toAccAddr := sdk.AccAddress(toEvmAddr.Bytes())
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)

	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	toBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	toLiquidBefore := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, liquidDenom).Amount

	amount := sdkmath.NewInt(1_000_000)

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, fromEvmAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, fromEvmAddr, contract, s.network.GetStateDB(), &method, []any{
		fromEvmAddr, toEvmAddr, amount.BigInt(),
	})
	s.Require().NoError(err)

	fromBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	toBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, utils.BaseDenom).Amount
	moduleBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	toLiquidAfter := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, liquidDenom).Amount

	s.Require().Equal(amount, fromBaseBefore.Sub(fromBaseAfter),
		"liquidateFrom base must decrease by exactly the liquidated amount on the EOA path")
	s.Require().Equal(toBaseBefore, toBaseAfter,
		"liquidateTo base must NOT change (it only receives the liquid denom)")
	s.Require().Equal(amount, moduleBaseAfter.Sub(moduleBaseBefore),
		"module base must increase by exactly the liquidated amount")
	s.Require().Equal(amount, toLiquidAfter.Sub(toLiquidBefore),
		"liquidateTo must receive exactly the freshly-minted liquid amount")
	s.Require().Equal(amount, s.network.App.BankKeeper.GetSupply(ctx, liquidDenom).Amount,
		"liquid token supply must equal the freshly-minted amount; no extra liquid tokens may appear")
	s.Require().Equal(fromBaseBefore.Add(moduleBaseBefore), fromBaseAfter.Add(moduleBaseAfter),
		"base-denom conservation: from's debit must be fully accounted for by module's credit")
}

// TestRedeemEOASelfBankDeltas exercises the EOA self-redeem path
// (caller == origin AND redeemFrom == redeemTo). It is the EOA-side regression
// for the dedup logic in mirrorBankBaseDeltasIntoStateDB - on this path the
// mirror MUST be a no-op (because caller == origin), and the dedup never
// matters; but if a future change accidentally enables the mirror for
// caller==origin, this test would catch the resulting double-credit.
func (s *PrecompileTestSuite) TestRedeemEOASelfBankDeltas() {
	s.SetupTest()
	ctx := s.network.GetContext()

	fromEvmAddr := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(fromEvmAddr.Bytes())
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	liquidateAmount := sdkmath.NewInt(1_000_000)
	redeemAmount := sdkmath.NewInt(500_000)

	// Liquidate first so we have liquid tokens to redeem to ourselves.
	liquidateMethod := s.precompile.Methods[liquid.LiquidateMethod]
	liqContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, fromEvmAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, fromEvmAddr, liqContract, s.network.GetStateDB(), &liquidateMethod, []any{
		fromEvmAddr, fromEvmAddr, liquidateAmount.BigInt(),
	})
	s.Require().NoError(err)

	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	fromLiquidBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, liquidDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	supplyBefore := s.network.App.BankKeeper.GetSupply(ctx, liquidDenom).Amount

	redeemMethod := s.precompile.Methods[liquid.RedeemMethod]
	redContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, fromEvmAddr, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, fromEvmAddr, redContract, s.network.GetStateDB(), &redeemMethod, []any{
		fromEvmAddr, fromEvmAddr, liquidDenom, redeemAmount.BigInt(),
	})
	s.Require().NoError(err)

	fromBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	fromLiquidAfter := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, liquidDenom).Amount
	moduleBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	supplyAfter := s.network.App.BankKeeper.GetSupply(ctx, liquidDenom).Amount

	s.Require().Equal(redeemAmount, fromBaseAfter.Sub(fromBaseBefore),
		"self-redeem must credit redeemFrom exactly once (dedup regression: a double-mirror would show +2x)")
	s.Require().Equal(redeemAmount, fromLiquidBefore.Sub(fromLiquidAfter),
		"self-redeem must burn exactly the redeemed liquid amount from the holder")
	s.Require().Equal(redeemAmount, moduleBaseBefore.Sub(moduleBaseAfter),
		"module base must decrease by exactly the redeemed amount")
	s.Require().Equal(redeemAmount, supplyBefore.Sub(supplyAfter),
		"liquid supply must decrease by exactly the redeemed amount via burn")
	s.Require().Equal(fromBaseBefore.Add(moduleBaseBefore), fromBaseAfter.Add(moduleBaseAfter),
		"base-denom conservation: module's debit must be fully accounted for by self-redeemer's credit")
}

// TestRedeemEOACrossAccountBankDeltas exercises the EOA cross-account redeem
// path (caller == origin, redeemFrom != redeemTo). It pins down that on the
// EOA path the keeper still routes the principal to redeemTo, redeemFrom
// loses only the liquid denom, and no extra entries leak in either direction.
func (s *PrecompileTestSuite) TestRedeemEOACrossAccountBankDeltas() {
	s.SetupTest()
	ctx := s.network.GetContext()

	fromEvmAddr := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(fromEvmAddr.Bytes())
	toEvmAddr := s.keyring.GetAddr(1)
	toAccAddr := sdk.AccAddress(toEvmAddr.Bytes())
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	liquidateAmount := sdkmath.NewInt(1_000_000)
	redeemAmount := sdkmath.NewInt(400_000)

	// Liquidate the vesting principal to fromEvmAddr so we have aLIQUID0
	// available on the redeemFrom side.
	liquidateMethod := s.precompile.Methods[liquid.LiquidateMethod]
	liqContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, fromEvmAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, fromEvmAddr, liqContract, s.network.GetStateDB(), &liquidateMethod, []any{
		fromEvmAddr, fromEvmAddr, liquidateAmount.BigInt(),
	})
	s.Require().NoError(err)

	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	fromLiquidBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, liquidDenom).Amount
	toBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	supplyBefore := s.network.App.BankKeeper.GetSupply(ctx, liquidDenom).Amount

	redeemMethod := s.precompile.Methods[liquid.RedeemMethod]
	redContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, fromEvmAddr, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, fromEvmAddr, redContract, s.network.GetStateDB(), &redeemMethod, []any{
		fromEvmAddr, toEvmAddr, liquidDenom, redeemAmount.BigInt(),
	})
	s.Require().NoError(err)

	fromBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	fromLiquidAfter := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, liquidDenom).Amount
	toBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, utils.BaseDenom).Amount
	moduleBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	supplyAfter := s.network.App.BankKeeper.GetSupply(ctx, liquidDenom).Amount

	s.Require().Equal(fromBaseBefore, fromBaseAfter,
		"cross-account redeem must NOT change redeemFrom base (only the liquid denom moves on this side)")
	s.Require().Equal(redeemAmount, fromLiquidBefore.Sub(fromLiquidAfter),
		"redeemFrom must lose exactly the redeemed liquid amount")
	s.Require().Equal(redeemAmount, toBaseAfter.Sub(toBaseBefore),
		"redeemTo base must increase by exactly the redeemed amount on the EOA cross-account path")
	s.Require().Equal(redeemAmount, moduleBaseBefore.Sub(moduleBaseAfter),
		"module base must decrease by exactly the redeemed amount")
	s.Require().Equal(redeemAmount, supplyBefore.Sub(supplyAfter),
		"liquid supply must decrease by exactly the redeemed amount via burn")
	s.Require().Equal(moduleBaseBefore.Add(toBaseBefore), moduleBaseAfter.Add(toBaseAfter),
		"base-denom conservation: module's debit must be fully accounted for by redeemTo's credit")
}

// ---------------------------------------------------------------------------
// Tier 2 - keeper edge cases that strengthen integration coverage.
//
// These exercise specific x/liquidvesting branches that integration tests
// only touch implicitly: liquidating exactly the locked total (account must
// be preserved as an empty vesting account, not destroyed), re-minting a
// liquid denom after a full redeem deletes the previous one, multi-denom
// locked vesting schedules (only the targeted denom is affected),
// liquidate-while-vesting-still-ongoing, liquidate-after-lockup-expired,
// and the EOA self-target (from == to) bank-delta path that is otherwise
// only verified through the Safe.
// ---------------------------------------------------------------------------

// TestLiquidateExactTotalLockedAccountPreserved verifies that liquidating
// exactly the entire locked vesting principal:
//  1. moves all base coins out of the from-account into the module;
//  2. mints exactly that amount of a new aLIQUID denom into the to-account;
//  3. PRESERVES the auth account as a ClawbackVestingAccount (it must NOT
//     be replaced by a plain account or destroyed) with zeroed schedules;
//  4. leaves the account in a state where any subsequent Liquidate fails
//     because there is nothing locked left.
//
// This is the boundary case for x/liquidvesting/types.SubtractAmountFromPeriods
// when subtrahend == total - the schedule is preserved but with empty amounts.
func (s *PrecompileTestSuite) TestLiquidateExactTotalLockedAccountPreserved() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)
	toAccAddr := sdk.AccAddress(to.Bytes())
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	totalLocked := vestingAmount.AmountOf(utils.BaseDenom)

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, totalLocked.BigInt(),
	})
	s.Require().NoError(err, "liquidating exactly the locked total must succeed")

	s.Require().Equal(sdkmath.ZeroInt(),
		s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount,
		"from-account base must be fully drained")
	s.Require().Equal(totalLocked,
		s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"module must receive the full locked total")
	s.Require().Equal(totalLocked,
		s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, liquidDenom).Amount,
		"to-account must receive the full liquid mint")

	acc := s.network.App.AccountKeeper.GetAccount(ctx, fromAccAddr)
	s.Require().NotNil(acc, "auth account must still exist after full liquidation (account preservation)")
	cba, ok := acc.(*vestingtypes.ClawbackVestingAccount)
	s.Require().True(ok, "auth account type must be preserved as ClawbackVestingAccount, not downgraded")
	s.Require().True(cba.OriginalVesting.AmountOf(utils.BaseDenom).IsZero(),
		"OriginalVesting must be zero after liquidating the entire principal")
	s.Require().True(cba.LockedCoins(ctx.BlockTime()).AmountOf(utils.BaseDenom).IsZero(),
		"LockedCoins must be zero after liquidating the entire principal")

	// A follow-up Liquidate must fail: nothing left locked, target denom
	// is no longer present in GetLockedUpCoins.
	contract2, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err = s.precompile.Liquidate(ctx, from, contract2, s.network.GetStateDB(), &method, []any{
		from, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err, "Liquidate against a fully-drained vesting account must fail")
	s.Require().ErrorContains(err, "doesn't contain coin specified as liquidation target",
		"empty-schedule error must propagate verbatim")
}

// TestLiquidateAfterFullRedeemCreatesNewTokenPair verifies that after a
// liquid denom is fully redeemed (which deletes the keeper's Denom record
// and toggles off the ERC20 conversion), the next Liquidate creates an
// entirely NEW liquid denom (counter is monotonic) bound to a distinct
// ERC20 contract address. This pins down two independent invariants:
//   - the deleted denom is NOT recycled;
//   - the keeper does NOT silently re-bind the same ERC20 address.
func (s *PrecompileTestSuite) TestLiquidateAfterFullRedeemCreatesNewTokenPair() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	liquidateMethod := s.precompile.Methods[liquid.LiquidateMethod]
	redeemMethod := s.precompile.Methods[liquid.RedeemMethod]

	// First Liquidate: aLIQUID0, capture erc20 address #1.
	c1, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	bz, err := s.precompile.Liquidate(ctx, from, c1, s.network.GetStateDB(), &liquidateMethod, []any{
		from, from, big.NewInt(1_000_000),
	})
	s.Require().NoError(err)
	out, err := liquidateMethod.Outputs.Unpack(bz)
	s.Require().NoError(err)
	erc20AddrFirst, ok := out[1].(common.Address)
	s.Require().True(ok)
	s.Require().NotEqual(common.Address{}, erc20AddrFirst, "first ERC20 address must be non-zero")

	denomFirst := liquidtypes.DenomBaseNameFromID(0)
	_, found := s.network.App.LiquidVestingKeeper.GetDenom(ctx, denomFirst)
	s.Require().True(found, "first liquid denom must be registered after Liquidate")

	// Full Redeem of aLIQUID0: keeper deletes the Denom and toggles off
	// the ERC20 conversion.
	c2, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, from, c2, s.network.GetStateDB(), &redeemMethod, []any{
		from, from, denomFirst, big.NewInt(1_000_000),
	})
	s.Require().NoError(err)

	_, found = s.network.App.LiquidVestingKeeper.GetDenom(ctx, denomFirst)
	s.Require().False(found, "first liquid denom must be deleted after full redeem")

	// Second Liquidate: counter went 0 -> 1 on the first call, so the
	// new mint is aLIQUID1 (NOT aLIQUID0 - the deleted denom is not recycled).
	c3, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	bz, err = s.precompile.Liquidate(ctx, from, c3, s.network.GetStateDB(), &liquidateMethod, []any{
		from, from, big.NewInt(1_000_000),
	})
	s.Require().NoError(err)
	out, err = liquidateMethod.Outputs.Unpack(bz)
	s.Require().NoError(err)
	erc20AddrSecond, ok := out[1].(common.Address)
	s.Require().True(ok)

	denomSecond := liquidtypes.DenomBaseNameFromID(1)
	_, found = s.network.App.LiquidVestingKeeper.GetDenom(ctx, denomSecond)
	s.Require().True(found, "second liquid denom (aLIQUID1) must be registered, not the recycled aLIQUID0")

	s.Require().NotEqual(erc20AddrFirst, erc20AddrSecond,
		"new liquid denom must bind to a fresh ERC20 contract address (no silent re-bind)")
}

// TestLiquidateMultiDenomLockedVesting verifies that Liquidate operates only
// on the targeted base denom of the schedule and leaves any other locked
// denoms untouched. We construct a vesting account with 3 lockup periods
// each holding aISLM AND a sibling test denom; after liquidating aISLM, the
// sibling denom must be unchanged in both bank balance and schedule.
func (s *PrecompileTestSuite) TestLiquidateMultiDenomLockedVesting() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)
	toAccAddr := sdk.AccAddress(to.Bytes())

	const siblingDenom = "atoken"

	// Lockup schedule: 3 periods, each holding 1_000_000 aISLM + 100_000 atoken.
	periodCoins := sdk.NewCoins(
		sdk.NewInt64Coin(utils.BaseDenom, 1_000_000),
		sdk.NewInt64Coin(siblingDenom, 100_000),
	)
	totalCoins := sdk.NewCoins(
		sdk.NewInt64Coin(utils.BaseDenom, 3_000_000),
		sdk.NewInt64Coin(siblingDenom, 300_000),
	)
	lockupPeriods := sdkvesting.Periods{
		{Length: 100000, Amount: periodCoins},
		{Length: 100000, Amount: periodCoins},
		{Length: 100000, Amount: periodCoins},
	}
	vestingPeriods := sdkvesting.Periods{
		{Length: 0, Amount: totalCoins}, // instantly vested - only lockup matters
	}

	funder := sdk.AccAddress(liquidtypes.ModuleName)
	baseAccount := authtypes.NewBaseAccountWithAddress(fromAccAddr)
	baseAccount.AccountNumber = s.network.App.AccountKeeper.NextAccountNumber(ctx)
	startTime := ctx.BlockTime().Add(-10 * time.Second)
	cba := vestingtypes.NewClawbackVestingAccount(
		baseAccount, funder, totalCoins, startTime, lockupPeriods, vestingPeriods, nil,
	)
	s.Require().NoError(haqqtestutil.FundAccount(ctx, s.network.App.BankKeeper, fromAccAddr, totalCoins))
	s.network.App.AccountKeeper.SetAccount(ctx, cba)

	siblingBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, siblingDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, big.NewInt(1_000_000),
	})
	s.Require().NoError(err)

	// Bank: aISLM moved out, atoken untouched.
	s.Require().Equal(sdkmath.NewInt(2_000_000),
		s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount,
		"aISLM bank balance must drop by exactly the liquidated amount")
	s.Require().Equal(siblingBefore,
		s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, siblingDenom).Amount,
		"sibling denom bank balance must NOT change when liquidating aISLM")

	// Schedule: each lockup period must still hold 100_000 atoken; aISLM
	// per-period amount is reduced proportionally.
	updated := s.network.App.AccountKeeper.GetAccount(ctx, fromAccAddr).(*vestingtypes.ClawbackVestingAccount)
	s.Require().Equal(sdkmath.NewInt(300_000),
		updated.LockupPeriods.TotalAmount().AmountOf(siblingDenom),
		"sibling denom must remain fully locked across the schedule")
	s.Require().Equal(sdkmath.NewInt(2_000_000),
		updated.LockupPeriods.TotalAmount().AmountOf(utils.BaseDenom),
		"aISLM lockup total must drop by exactly the liquidated amount")

	// Liquid mint goes to the to-account in the freshly created denom.
	s.Require().Equal(sdkmath.NewInt(1_000_000),
		s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, liquidtypes.DenomBaseNameFromID(0)).Amount,
		"to-account must receive exactly 1_000_000 aLIQUID0")
}

// TestLiquidateVestingStillOngoing verifies the keeper rejects Liquidate
// when the from-account still has non-vested coins (vestingPeriods has a
// future-completing period). Without this guard, a user could pre-liquidate
// not-yet-vested principal.
func (s *PrecompileTestSuite) TestLiquidateVestingStillOngoing() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)

	// Vesting still ongoing: a single vesting period of 100_000s starting
	// 10s ago - GetVestingCoins(blockTime) returns the still-unvested coins.
	periods := sdkvesting.Periods{
		{Length: 100000, Amount: vestingAmount},
	}
	funder := sdk.AccAddress(liquidtypes.ModuleName)
	baseAccount := authtypes.NewBaseAccountWithAddress(fromAccAddr)
	baseAccount.AccountNumber = s.network.App.AccountKeeper.NextAccountNumber(ctx)
	startTime := ctx.BlockTime().Add(-10 * time.Second)
	cba := vestingtypes.NewClawbackVestingAccount(
		baseAccount, funder, vestingAmount, startTime, periods, periods, nil,
	)
	s.Require().NoError(haqqtestutil.FundAccount(ctx, s.network.App.BankKeeper, fromAccAddr, vestingAmount))
	s.network.App.AccountKeeper.SetAccount(ctx, cba)

	// Sanity: the account does report non-zero vesting coins at this time.
	s.Require().False(cba.GetVestingCoins(ctx.BlockTime()).IsZero(),
		"fixture invariant: vesting must still be ongoing at block time")

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "has vesting ongoing periods",
		"keeper must reject Liquidate while vesting is still ongoing")
}

// TestLiquidateLockupExpired verifies the keeper rejects Liquidate when the
// account's lockup has fully elapsed (GetLockedUpCoins returns zero for the
// target denom, so Find(amount.Denom) returns hasTargetDenom=false).
func (s *PrecompileTestSuite) TestLiquidateLockupExpired() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)

	// Lockup completed in the past: start time is far enough back that all
	// 3 short lockup periods have already elapsed by block time.
	lockupPeriods := sdkvesting.Periods{
		{Length: 10, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
		{Length: 10, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
		{Length: 10, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
	}
	vestingPeriods := sdkvesting.Periods{
		{Length: 0, Amount: vestingAmount},
	}

	funder := sdk.AccAddress(liquidtypes.ModuleName)
	baseAccount := authtypes.NewBaseAccountWithAddress(fromAccAddr)
	baseAccount.AccountNumber = s.network.App.AccountKeeper.NextAccountNumber(ctx)
	startTime := ctx.BlockTime().Add(-1 * time.Hour) // 3600s ago, far past the 30s of total lockup
	cba := vestingtypes.NewClawbackVestingAccount(
		baseAccount, funder, vestingAmount, startTime, lockupPeriods, vestingPeriods, nil,
	)
	s.Require().NoError(haqqtestutil.FundAccount(ctx, s.network.App.BankKeeper, fromAccAddr, vestingAmount))
	s.network.App.AccountKeeper.SetAccount(ctx, cba)

	// Sanity: lockup is fully expired at block time.
	s.Require().True(cba.GetLockedUpCoins(ctx.BlockTime()).IsZero(),
		"fixture invariant: lockup must be fully expired at block time")

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "doesn't contain coin specified as liquidation target",
		"keeper must reject Liquidate after lockup expiry (no target denom in locked coins)")
}

// TestLiquidateEOASameAddressBankDeltas exercises the EOA same-address
// (from == to) Liquidate path with explicit bank-delta assertions. The
// existing TestLiquidateSuccess uses different from/to; this fills the
// matching invariant for the self-target case.
func (s *PrecompileTestSuite) TestLiquidateEOASameAddressBankDeltas() {
	s.SetupTest()
	ctx := s.network.GetContext()

	addr := utiltx.GenerateAddress()
	accAddr := sdk.AccAddress(addr.Bytes())
	s.createClawbackVestingAccount(ctx, accAddr)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	amount := sdkmath.NewInt(1_000_000)

	baseBefore := s.network.App.BankKeeper.GetBalance(ctx, accAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, addr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, addr, contract, s.network.GetStateDB(), &method, []any{
		addr, addr, amount.BigInt(),
	})
	s.Require().NoError(err)

	baseAfter := s.network.App.BankKeeper.GetBalance(ctx, accAddr, utils.BaseDenom).Amount
	moduleBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	liquidAfter := s.network.App.BankKeeper.GetBalance(ctx, accAddr, liquidDenom).Amount

	s.Require().Equal(amount, baseBefore.Sub(baseAfter),
		"self-target Liquidate must debit the EOA base by exactly the liquidated amount")
	s.Require().Equal(amount, liquidAfter,
		"self-target Liquidate must credit the EOA with exactly the liquidated amount of aLIQUID0")
	s.Require().Equal(amount, moduleBaseAfter.Sub(moduleBaseBefore),
		"self-target Liquidate must credit the module by exactly the liquidated amount")
	s.Require().Equal(baseBefore.Add(moduleBaseBefore), baseAfter.Add(moduleBaseAfter),
		"base-denom conservation must hold for self-target EOA Liquidate")
}

// ---------------------------------------------------------------------------
// Authz edge cases: expired grants must be treated as missing, and the
// origin/sender invariant must be enforced uniformly across both EOA-caller
// and contract-caller paths so a contract caller cannot use authz to act on
// behalf of arbitrary senders that don't match origin.
// ---------------------------------------------------------------------------

// TestLiquidateAuthzExpired verifies the precompile treats an expired authz
// grant the same as a missing one. We save a grant with a 1-second expiration
// (NewGrant requires expiration > current block time), then advance the
// in-memory ctx block time past it before calling Liquidate. GetAuthorization
// must return nil and the precompile must surface the standard
// "does not exist or is expired" error.
func (s *PrecompileTestSuite) TestLiquidateAuthzExpired() {
	s.SetupTest()
	ctx := s.network.GetContext()

	originAddr := utiltx.GenerateAddress()
	originAccAddr := sdk.AccAddress(originAddr.Bytes())
	callerAddr := utiltx.GenerateAddress()
	callerAccAddr := sdk.AccAddress(callerAddr.Bytes())
	to := s.keyring.GetAddr(1)

	s.createClawbackVestingAccount(ctx, originAccAddr)

	// Grant with a 1-second expiration window starting from the current
	// block time. NewGrant rejects expirations that are already in the
	// past, so we cannot save an "already expired" grant directly.
	expiration := ctx.BlockTime().Add(time.Second)
	err := s.network.App.AuthzKeeper.SaveGrant(
		ctx, callerAccAddr, originAccAddr,
		sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{})),
		&expiration,
	)
	s.Require().NoError(err)

	// Sanity: at this block time the grant is still live.
	auth, _ := s.network.App.AuthzKeeper.GetAuthorization(
		ctx, callerAccAddr, originAccAddr, sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{}),
	)
	s.Require().NotNil(auth, "fixture invariant: grant must be live before time advance")

	// Advance the in-memory block time past the expiration. The next
	// GetAuthorization must report it as expired.
	ctx = ctx.WithBlockTime(expiration.Add(time.Second))
	auth, _ = s.network.App.AuthzKeeper.GetAuthorization(
		ctx, callerAccAddr, originAccAddr, sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{}),
	)
	s.Require().Nil(auth, "fixture invariant: grant must be expired after time advance")

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerAddr, s.precompile, 500000)
	_, err = s.precompile.Liquidate(ctx, originAddr, contract, s.network.GetStateDB(), &method, []any{
		originAddr, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "does not exist or is expired",
		"expired authz grant must produce the same error as a missing grant")
}

// TestRedeemAuthzExpired verifies the same expired-grant invariant for the
// Redeem method. Symmetric to TestLiquidateAuthzExpired - both methods share
// the same authz lookup path, so both must observe the time-based expiry.
func (s *PrecompileTestSuite) TestRedeemAuthzExpired() {
	s.SetupTest()
	ctx := s.network.GetContext()

	originAddr := utiltx.GenerateAddress()
	originAccAddr := sdk.AccAddress(originAddr.Bytes())
	callerAddr := utiltx.GenerateAddress()
	callerAccAddr := sdk.AccAddress(callerAddr.Bytes())

	expiration := ctx.BlockTime().Add(time.Second)
	err := s.network.App.AuthzKeeper.SaveGrant(
		ctx, callerAccAddr, originAccAddr,
		sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgRedeem{})),
		&expiration,
	)
	s.Require().NoError(err)

	ctx = ctx.WithBlockTime(expiration.Add(time.Second))

	method := s.precompile.Methods[liquid.RedeemMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerAddr, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, originAddr, contract, s.network.GetStateDB(), &method, []any{
		originAddr, originAddr, liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "does not exist or is expired",
		"expired Redeem authz grant must produce the same error as a missing grant")
}

// TestLiquidateThirdPartySenderViaContract pins down the origin/sender
// invariant on the contract-caller path: when the caller is a contract, the
// ABI from-field (sender) must equal either the caller (contract acting on
// its own behalf) OR the tx origin (contract acting via authz). A third
// party that is neither must be rejected BEFORE the authz lookup.
//
// Without this guard, a malicious contract could ask the precompile to
// execute MsgLiquidate.From = victim using its own authz from origin -
// trivially defeating the per-granter authorization model.
func (s *PrecompileTestSuite) TestLiquidateThirdPartySenderViaContract() {
	s.SetupTest()
	ctx := s.network.GetContext()

	originAddr := utiltx.GenerateAddress()
	originAccAddr := sdk.AccAddress(originAddr.Bytes())
	callerAddr := utiltx.GenerateAddress() // contract caller, != origin
	callerAccAddr := sdk.AccAddress(callerAddr.Bytes())
	thirdParty := utiltx.GenerateAddress() // ABI from-field, != caller AND != origin
	to := s.keyring.GetAddr(1)

	s.createClawbackVestingAccount(ctx, originAccAddr)

	// Save a valid authz grant from origin to caller. The check we want
	// to validate runs BEFORE the authz lookup, so the grant being
	// present just makes it an even more aggressive attempt.
	expiration := time.Now().Add(time.Hour)
	err := s.network.App.AuthzKeeper.SaveGrant(
		ctx, callerAccAddr, originAccAddr,
		sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{})),
		&expiration,
	)
	s.Require().NoError(err)

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerAddr, s.precompile, 500000)
	_, err = s.precompile.Liquidate(ctx, originAddr, contract, s.network.GetStateDB(), &method, []any{
		thirdParty, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "origin address",
		"sender (msg.from) must equal caller or origin even on the contract-caller path")
}

// TestRedeemThirdPartySenderViaContract is the symmetric test for Redeem:
// the same origin/sender check exists in Redeem and must be exercised on
// the contract-caller path.
func (s *PrecompileTestSuite) TestRedeemThirdPartySenderViaContract() {
	s.SetupTest()
	ctx := s.network.GetContext()

	originAddr := utiltx.GenerateAddress()
	originAccAddr := sdk.AccAddress(originAddr.Bytes())
	callerAddr := utiltx.GenerateAddress()
	callerAccAddr := sdk.AccAddress(callerAddr.Bytes())
	thirdParty := utiltx.GenerateAddress()

	expiration := time.Now().Add(time.Hour)
	err := s.network.App.AuthzKeeper.SaveGrant(
		ctx, callerAccAddr, originAccAddr,
		sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgRedeem{})),
		&expiration,
	)
	s.Require().NoError(err)

	method := s.precompile.Methods[liquid.RedeemMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerAddr, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, originAddr, contract, s.network.GetStateDB(), &method, []any{
		thirdParty, thirdParty, liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "origin address",
		"sender (msg.from) must equal caller or origin even on the contract-caller path (Redeem)")
}

// ---------------------------------------------------------------------------
// Keeper-side error pass-through: invariant that liquidvesting keeper errors
// are surfaced verbatim through the precompile, no side effects leak into
// bank, and the precompile does not panic / partially commit.
//
// These tests exercise the EOA path on purpose: the precompile's RunAtomic
// rollback is bypassed when methods are invoked directly (no Run dispatcher),
// so the keeper itself must short-circuit BEFORE any state mutation.
// Each test asserts (a) the error contains the expected SDK message and
// (b) bank state on the touched accounts is unchanged.
// ---------------------------------------------------------------------------

// TestLiquidateOnRegularAccount verifies the keeper rejects Liquidate with
// "is regular nothing to liquidate" when the from-account is not a
// ClawbackVestingAccount, and that no state mutation occurs.
func (s *PrecompileTestSuite) TestLiquidateOnRegularAccount() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := s.keyring.GetAddr(0)
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)

	// Sanity: the keyring account exists but is NOT a vesting account.
	acc := s.network.App.AccountKeeper.GetAccount(ctx, fromAccAddr)
	s.Require().NotNil(acc)
	_, isVesting := acc.(*vestingtypes.ClawbackVestingAccount)
	s.Require().False(isVesting, "test fixture must be a plain account, not a vesting account")

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "is regular nothing to liquidate",
		"keeper error must be propagated verbatim through the precompile")

	s.Require().Equal(fromBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount,
		"failed Liquidate must not touch the from-account base balance")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"failed Liquidate must not credit the liquid vesting module")
}

// TestLiquidateModuleDisabled verifies the keeper short-circuits with
// "liquid vesting module is disabled" when params.EnableLiquidVesting is
// false, before touching any accounts.
func (s *PrecompileTestSuite) TestLiquidateModuleDisabled() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	// Disable the module via params, refresh ctx so Liquidate observes it.
	s.network.App.LiquidVestingKeeper.SetLiquidVestingEnabled(ctx, false)
	ctx = s.network.GetContext()

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "liquid vesting module is disabled",
		"disabled-module guard must short-circuit Liquidate at the keeper level")

	s.Require().Equal(fromBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount,
		"Liquidate against a disabled module must not move funds")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"Liquidate against a disabled module must not credit the module")
}

// TestLiquidateBelowMinimum verifies the keeper rejects Liquidate when the
// amount is strictly below MinimumLiquidationAmount (set to 1_000_000 in the
// test fixture). 999_999 must trip the guard; 1_000_000 must succeed (the
// boundary success case is covered by other tests).
func (s *PrecompileTestSuite) TestLiquidateBelowMinimum() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, big.NewInt(999_999),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "unable to liquidate amount lesser than",
		"sub-minimum amount must be rejected by the keeper")

	s.Require().Equal(fromBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount,
		"sub-minimum Liquidate must not move funds")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"sub-minimum Liquidate must not credit the module")
}

// TestLiquidateExceedsLocked verifies the keeper rejects Liquidate when the
// requested amount exceeds the account's currently-locked balance for the
// target denom. Vesting fixture has 3_000_000 locked; we ask for 4_000_000.
func (s *PrecompileTestSuite) TestLiquidateExceedsLocked() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, big.NewInt(4_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "doesn't have sufficient amount of target coin for liquidation",
		"over-locked Liquidate must be rejected by the keeper")

	s.Require().Equal(fromBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount,
		"over-locked Liquidate must not move funds")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"over-locked Liquidate must not credit the module")
}

// TestRedeemUnknownDenom verifies the keeper rejects Redeem when the supplied
// denom has no corresponding x/liquidvesting Denom record.
func (s *PrecompileTestSuite) TestRedeemUnknownDenom() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := s.keyring.GetAddr(0)
	fromAccAddr := sdk.AccAddress(from.Bytes())

	// Use a denom id far above any aLIQUID we could have created in this
	// test (counter starts at 0); guarantees GetDenom returns not-found.
	unknownDenom := liquidtypes.DenomBaseNameFromID(9999)

	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.RedeemMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Redeem(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, from, unknownDenom, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "does not exist",
		"unknown liquid denom must be rejected at the keeper denom lookup")

	s.Require().Equal(fromBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount,
		"failed Redeem must not change the from-account base balance")
}

// TestRedeemInsufficientLiquidBalance verifies the keeper rejects Redeem when
// the from-account holds fewer liquid tokens than requested. We mint
// 1_000_000 via Liquidate then attempt to redeem 2_000_000.
func (s *PrecompileTestSuite) TestRedeemInsufficientLiquidBalance() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	liquidateMethod := s.precompile.Methods[liquid.LiquidateMethod]
	liqContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, liqContract, s.network.GetStateDB(), &liquidateMethod, []any{
		from, from, big.NewInt(1_000_000),
	})
	s.Require().NoError(err)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	fromLiquidBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, liquidDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	supplyBefore := s.network.App.BankKeeper.GetSupply(ctx, liquidDenom).Amount

	redeemMethod := s.precompile.Methods[liquid.RedeemMethod]
	redContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, from, redContract, s.network.GetStateDB(), &redeemMethod, []any{
		from, from, liquidDenom, big.NewInt(2_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "insufficient balance",
		"over-balance Redeem must be rejected by the keeper insufficient-balance check")

	s.Require().Equal(fromBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount,
		"failed Redeem must not change the from base balance")
	s.Require().Equal(fromLiquidBefore, s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, liquidDenom).Amount,
		"failed Redeem must not burn any liquid tokens")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"failed Redeem must not move base from the module")
	s.Require().Equal(supplyBefore, s.network.App.BankKeeper.GetSupply(ctx, liquidDenom).Amount,
		"failed Redeem must not change liquid supply")
}

// TestLiquidateEOAvsContractParity verifies that a Liquidate call produces the
// SAME post-call bank state regardless of whether the precompile is invoked
// directly by the EOA holder (caller == origin, mirror skipped) or from a
// contract caller via authz (caller != origin, mirror MUST fire to keep
// StateDB and bank consistent). If the mirror were silently broken on the
// contract path, or wrongly enabled on the EOA path, the two scenarios would
// diverge and this test would catch it.
//
// Note: each scenario is run in a freshly-set-up network, so addresses
// generated by utiltx are different across runs - we compare deltas, not
// absolute balances.
func (s *PrecompileTestSuite) TestLiquidateEOAvsContractParity() {
	type bankDelta struct {
		fromBaseDebit  sdkmath.Int
		toLiquidCredit sdkmath.Int
		moduleBaseAdd  sdkmath.Int
		liquidSupply   sdkmath.Int
	}

	amount := sdkmath.NewInt(1_000_000)
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)

	// runScenario performs a Liquidate from a fresh vesting account into a
	// fresh receiver and returns the observed bank deltas. The caller flag
	// flips between caller == origin (EOA path, no authz) and caller !=
	// origin (contract path, authz required).
	runScenario := func(callerIsOrigin bool) bankDelta {
		s.SetupTest()
		ctx := s.network.GetContext()

		fromEvmAddr := utiltx.GenerateAddress()
		fromAccAddr := sdk.AccAddress(fromEvmAddr.Bytes())
		toEvmAddr := utiltx.GenerateAddress()
		toAccAddr := sdk.AccAddress(toEvmAddr.Bytes())

		s.createClawbackVestingAccount(ctx, fromAccAddr)

		callerEvmAddr := fromEvmAddr
		if !callerIsOrigin {
			callerEvmAddr = utiltx.GenerateAddress()
			callerAccAddr := sdk.AccAddress(callerEvmAddr.Bytes())
			expiration := time.Now().Add(time.Hour)
			err := s.network.App.AuthzKeeper.SaveGrant(
				ctx,
				callerAccAddr,
				fromAccAddr,
				sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{})),
				&expiration,
			)
			s.Require().NoError(err)
		}

		fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
		toLiquidBefore := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, liquidDenom).Amount
		moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

		method := s.precompile.Methods[liquid.LiquidateMethod]
		contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerEvmAddr, s.precompile, 500000)
		_, err := s.precompile.Liquidate(ctx, fromEvmAddr, contract, s.network.GetStateDB(), &method, []any{
			fromEvmAddr, toEvmAddr, amount.BigInt(),
		})
		s.Require().NoError(err)

		fromBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
		toLiquidAfter := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, liquidDenom).Amount
		moduleBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

		return bankDelta{
			fromBaseDebit:  fromBaseBefore.Sub(fromBaseAfter),
			toLiquidCredit: toLiquidAfter.Sub(toLiquidBefore),
			moduleBaseAdd:  moduleBaseAfter.Sub(moduleBaseBefore),
			liquidSupply:   s.network.App.BankKeeper.GetSupply(ctx, liquidDenom).Amount,
		}
	}

	eoa := runScenario(true)
	contract := runScenario(false)

	s.Require().Equal(amount, eoa.fromBaseDebit, "EOA scenario: from must be debited by exactly the liquidated amount")
	s.Require().Equal(amount, contract.fromBaseDebit, "contract scenario: from must be debited by exactly the liquidated amount")
	s.Require().Equal(eoa.fromBaseDebit, contract.fromBaseDebit,
		"from base debit must be identical between EOA and contract caller paths (mirror parity)")
	s.Require().Equal(eoa.toLiquidCredit, contract.toLiquidCredit,
		"to liquid credit must be identical between EOA and contract caller paths")
	s.Require().Equal(eoa.moduleBaseAdd, contract.moduleBaseAdd,
		"module base credit must be identical between EOA and contract caller paths")
	s.Require().Equal(eoa.liquidSupply, contract.liquidSupply,
		"final liquid supply must be identical between EOA and contract caller paths")
}

// ---------------------------------------------------------------------------
// Tier 3 - defensive / boundary tests for user-controlled accounts
// (EOA + smart-contract callers). Module-account-as-from is intentionally
// out of scope.
//
// Each test exercises a single defensive guard and asserts:
//   (a) the precompile returns the expected error class without panicking;
//   (b) bank state on the touched accounts is unchanged on failure paths.
// ---------------------------------------------------------------------------

// TestLiquidateAtExactMinimumAmount pins the boundary of the
// MinimumLiquidationAmount param: keeper rejects via amount.IsLT(min), so an
// amount EQUAL to the minimum must succeed (boundary is inclusive on the low
// side). If a future refactor flips the comparison to amount.LTE, this test
// will catch the regression by failing the success path.
func (s *PrecompileTestSuite) TestLiquidateAtExactMinimumAmount() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	to := s.keyring.GetAddr(1)
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	// Sanity: the configured param must equal the amount we are about to
	// liquidate; otherwise the boundary assertion below loses meaning.
	min := s.network.App.LiquidVestingKeeper.GetParams(ctx).MinimumLiquidationAmount
	s.Require().Equal(sdkmath.NewInt(1_000_000), min,
		"test setup must pin MinimumLiquidationAmount to 1_000_000")

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, to, min.BigInt(),
	})
	s.Require().NoError(err, "amount equal to MinimumLiquidationAmount must be accepted (boundary inclusive)")

	fromBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	moduleBaseAfter := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount
	s.Require().Equal(min, fromBaseBefore.Sub(fromBaseAfter),
		"from must be debited exactly the minimum amount")
	s.Require().Equal(min, moduleBaseAfter.Sub(moduleBaseBefore),
		"module must be credited exactly the minimum amount")
}

// TestLiquidateFromZeroAddress verifies the keeper rejects Liquidate when
// the from-account does not exist on chain. A fresh setup has no account at
// 0x0, so accountKeeper.GetAccount returns nil and the keeper aborts with
// "account does not exist". No bank movement may occur.
//
// NOTE: origin must match sender (the ABI from-field), so origin is also
// set to 0x0 here. This is impossible in production (no signed EVM tx can
// have origin == 0x0) but the precompile entry-point itself does not - and
// should not - enforce that, so the keeper guard is the actual defence.
func (s *PrecompileTestSuite) TestLiquidateFromZeroAddress() {
	s.SetupTest()
	ctx := s.network.GetContext()

	zeroAddr := common.Address{}
	zeroAccAddr := sdk.AccAddress(zeroAddr.Bytes())
	to := s.keyring.GetAddr(1)
	toAccAddr := sdk.AccAddress(to.Bytes())

	// Sanity: there must be no account at 0x0 in a fresh network.
	s.Require().Nil(s.network.App.AccountKeeper.GetAccount(ctx, zeroAccAddr),
		"fresh network must not have an account at the zero address")

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	zeroBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, zeroAccAddr, utils.BaseDenom).Amount
	toBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, zeroAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, zeroAddr, contract, s.network.GetStateDB(), &method, []any{
		zeroAddr, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "does not exist",
		"non-existent from-account must be rejected with an account-lookup error")

	s.Require().Equal(zeroBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, zeroAccAddr, utils.BaseDenom).Amount,
		"failed Liquidate must not touch the zero-address bank balance")
	s.Require().Equal(toBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, toAccAddr, utils.BaseDenom).Amount,
		"failed Liquidate must not touch the receiver bank balance")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"failed Liquidate must not credit the module")
}

// TestRedeemFromZeroAddress is the symmetric defence for Redeem: a non-
// existent from-account is rejected by the keeper before any bank movement.
func (s *PrecompileTestSuite) TestRedeemFromZeroAddress() {
	s.SetupTest()
	ctx := s.network.GetContext()

	zeroAddr := common.Address{}
	zeroAccAddr := sdk.AccAddress(zeroAddr.Bytes())

	s.Require().Nil(s.network.App.AccountKeeper.GetAccount(ctx, zeroAccAddr),
		"fresh network must not have an account at the zero address")

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	zeroBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, zeroAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.RedeemMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, zeroAddr, s.precompile, 500000)
	_, err := s.precompile.Redeem(ctx, zeroAddr, contract, s.network.GetStateDB(), &method, []any{
		zeroAddr, zeroAddr, liquidtypes.DenomBaseNameFromID(0), big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "does not exist",
		"non-existent from-account must be rejected with an account-lookup error")

	s.Require().Equal(zeroBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, zeroAccAddr, utils.BaseDenom).Amount,
		"failed Redeem must not touch the zero-address bank balance")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"failed Redeem must not touch the module bank balance")
}

// TestLiquidateAuthzWrongMsgType verifies that authz scope is enforced by
// message-type URL: a grant for MsgRedeem cannot authorize a Liquidate call
// from the same caller. The precompile must reject with
// ErrAuthzDoesNotExistOrExpired (not silently fall through), and no bank
// movement may occur.
func (s *PrecompileTestSuite) TestLiquidateAuthzWrongMsgType() {
	s.SetupTest()
	ctx := s.network.GetContext()

	originAddr := utiltx.GenerateAddress()
	originAccAddr := sdk.AccAddress(originAddr.Bytes())
	callerAddr := utiltx.GenerateAddress()
	callerAccAddr := sdk.AccAddress(callerAddr.Bytes())
	to := s.keyring.GetAddr(1)

	s.createClawbackVestingAccount(ctx, originAccAddr)

	// Wrong-type grant: MsgRedeem cannot authorize MsgLiquidate.
	expiration := time.Now().Add(time.Hour)
	err := s.network.App.AuthzKeeper.SaveGrant(
		ctx, callerAccAddr, originAccAddr,
		sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgRedeem{})),
		&expiration,
	)
	s.Require().NoError(err)

	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, originAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerAddr, s.precompile, 500000)
	_, err = s.precompile.Liquidate(ctx, originAddr, contract, s.network.GetStateDB(), &method, []any{
		originAddr, to, big.NewInt(1_000_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "authorization",
		"authz grant scoped to a different msg type must not authorize Liquidate")

	s.Require().Equal(fromBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, originAccAddr, utils.BaseDenom).Amount,
		"rejected wrong-type-grant Liquidate must not move from-base")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"rejected wrong-type-grant Liquidate must not credit the module")
}

// TestRedeemAuthzWrongMsgType is the symmetric authz-scope defence for
// Redeem: a grant for MsgLiquidate cannot authorize a Redeem call from the
// same caller. Setup needs an actual liquid denom in flight, so we first do
// a self-Liquidate via the EOA path (no authz needed) to mint aLIQUID0,
// then attempt the Redeem via the contract path with the wrong-type grant.
func (s *PrecompileTestSuite) TestRedeemAuthzWrongMsgType() {
	s.SetupTest()
	ctx := s.network.GetContext()

	originAddr := utiltx.GenerateAddress()
	originAccAddr := sdk.AccAddress(originAddr.Bytes())
	callerAddr := utiltx.GenerateAddress()
	callerAccAddr := sdk.AccAddress(callerAddr.Bytes())

	s.createClawbackVestingAccount(ctx, originAccAddr)

	// Step A: EOA self-Liquidate to obtain a real aLIQUID0 balance on origin.
	liqMethod := s.precompile.Methods[liquid.LiquidateMethod]
	liqContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, originAddr, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, originAddr, liqContract, s.network.GetStateDB(), &liqMethod, []any{
		originAddr, originAddr, big.NewInt(1_000_000),
	})
	s.Require().NoError(err, "setup self-Liquidate must succeed to mint aLIQUID0")

	// Step B: wrong-type grant - MsgLiquidate cannot authorize MsgRedeem.
	expiration := time.Now().Add(time.Hour)
	err = s.network.App.AuthzKeeper.SaveGrant(
		ctx, callerAccAddr, originAccAddr,
		sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{})),
		&expiration,
	)
	s.Require().NoError(err)

	denom := liquidtypes.DenomBaseNameFromID(0)
	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
	fromLiquidBefore := s.network.App.BankKeeper.GetBalance(ctx, originAccAddr, denom).Amount
	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, originAccAddr, utils.BaseDenom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	method := s.precompile.Methods[liquid.RedeemMethod]
	redeemContract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerAddr, s.precompile, 500000)
	_, err = s.precompile.Redeem(ctx, originAddr, redeemContract, s.network.GetStateDB(), &method, []any{
		originAddr, originAddr, denom, big.NewInt(500_000),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "authorization",
		"authz grant scoped to a different msg type must not authorize Redeem")

	s.Require().Equal(fromLiquidBefore, s.network.App.BankKeeper.GetBalance(ctx, originAccAddr, denom).Amount,
		"rejected wrong-type-grant Redeem must not burn liquid tokens")
	s.Require().Equal(fromBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, originAccAddr, utils.BaseDenom).Amount,
		"rejected wrong-type-grant Redeem must not return base coins to from")
	s.Require().Equal(moduleBaseBefore, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount,
		"rejected wrong-type-grant Redeem must not move the module's base balance")
}

// TestLiquidateToZeroAddress is a PIN test: it documents the current
// behaviour of Liquidate when the receiver of the freshly-minted aLIQUID*
// is the zero address (0x0). The precompile itself does not validate the
// receiver address, so the actual fate is decided by the bank module's
// SendCoinsFromModuleToAccount call inside the keeper.
//
// Expected current behaviour: the call SUCCEEDS - 0x0 is a normal,
// non-blocked bank holder, the aLIQUID0 supply is created on it, and from
// is correctly debited the base amount. The minted tokens are functionally
// unrecoverable (no key controls 0x0), but that is a caller-side concern
// and not a precompile-level guard.
//
// If this test ever starts FAILING, that means new validation has been
// added (e.g. blocked-address list, explicit zero check) - the change must
// be reviewed deliberately rather than silently accepted, hence pinning
// here.
func (s *PrecompileTestSuite) TestLiquidateToZeroAddress() {
	s.SetupTest()
	ctx := s.network.GetContext()

	from := utiltx.GenerateAddress()
	fromAccAddr := sdk.AccAddress(from.Bytes())
	zeroAddr := common.Address{}
	zeroAccAddr := sdk.AccAddress(zeroAddr.Bytes())
	s.createClawbackVestingAccount(ctx, fromAccAddr)

	denom := liquidtypes.DenomBaseNameFromID(0)
	moduleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)

	fromBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount
	zeroLiquidBefore := s.network.App.BankKeeper.GetBalance(ctx, zeroAccAddr, denom).Amount
	moduleBaseBefore := s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount

	amount := sdkmath.NewInt(1_000_000)
	method := s.precompile.Methods[liquid.LiquidateMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, from, s.precompile, 500000)
	_, err := s.precompile.Liquidate(ctx, from, contract, s.network.GetStateDB(), &method, []any{
		from, zeroAddr, amount.BigInt(),
	})
	s.Require().NoError(err,
		"PIN: Liquidate to 0x0 currently succeeds; if this fails a new receiver-side guard was added and must be reviewed")

	s.Require().Equal(amount, fromBaseBefore.Sub(s.network.App.BankKeeper.GetBalance(ctx, fromAccAddr, utils.BaseDenom).Amount),
		"PIN: from must be debited exactly the liquidated amount")
	s.Require().Equal(amount, s.network.App.BankKeeper.GetBalance(ctx, zeroAccAddr, denom).Amount.Sub(zeroLiquidBefore),
		"PIN: zero-address must receive the freshly-minted liquid supply")
	s.Require().Equal(amount, s.network.App.BankKeeper.GetBalance(ctx, moduleAddr, utils.BaseDenom).Amount.Sub(moduleBaseBefore),
		"PIN: module must be credited exactly the liquidated base amount")
	s.Require().Equal(amount, s.network.App.BankKeeper.GetSupply(ctx, denom).Amount,
		"PIN: aLIQUID0 total supply must equal the minted amount")
}
