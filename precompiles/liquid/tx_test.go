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
