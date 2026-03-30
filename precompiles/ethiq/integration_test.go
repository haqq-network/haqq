package ethiq_test

import (
	"math/big"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

// appID6FromAddrBech32 is the from-address of application ID 6 in the waitlist.
const appID6FromAddrBech32 = "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z"

// ---------------------------------------------------------------------------
// TestMintHaqqDirectly tests MintHaqq when called directly from an EOA
// (contract.CallerAddress == origin == sender).
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestMintHaqqDirectly() {
	s.SetupTest()
	ctx := s.network.GetContext()

	sender := s.keyring.GetAddr(0)
	receiver := s.keyring.GetAddr(1)
	burnAmount := big.NewInt(1e18)

	// Fund sender with enough aISLM to burn
	senderAccAddr := sdk.AccAddress(sender.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewIntFromBigInt(burnAmount).MulRaw(2)))
	err := s.network.App.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins)
	s.Require().NoError(err)
	err = s.network.App.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, senderAccAddr, coins)
	s.Require().NoError(err)

	method := s.precompile.Methods[ethiq.MintHaqq]

	// contract.CallerAddress == sender == origin → no authz required
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, sender, s.precompile, 200000)

	bz, err := s.precompile.MintHaqq(ctx, sender, contract, s.network.GetStateDB(), &method, []any{
		sender,
		receiver,
		burnAmount,
	})
	s.Require().NoError(err)
	s.Require().NotNil(bz)

	// Verify return value is a non-zero haqqAmount
	result := new(big.Int).SetBytes(bz)
	s.Require().True(result.Sign() > 0, "expected positive haqqAmount in return value")
}

// ---------------------------------------------------------------------------
// TestMintHaqqViaThirdPartyContract tests MintHaqq called through a "dummy"
// smart contract (contract.CallerAddress != origin).
//
// Flow:
//  1. granter (EOA) grants MintHaqqAuthorization to a dummy contract address.
//  2. The dummy contract calls MintHaqq on behalf of the granter.
//  3. The grant is consumed (deleted) after MintHaqq.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestMintHaqqViaThirdPartyContract() {
	s.SetupTest()
	ctx := s.network.GetContext()

	// The EOA that owns the funds and creates the authorization
	origin := s.keyring.GetAddr(0)
	receiver := s.keyring.GetAddr(1)
	burnAmount := big.NewInt(1e18)

	// A dummy "contract" address acting as the third-party caller
	dummyContract := utiltx.GenerateAddress()

	// Fund the origin (sender) with aISLM
	originAccAddr := sdk.AccAddress(origin.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewIntFromBigInt(burnAmount).MulRaw(2)))
	err := s.network.App.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins)
	s.Require().NoError(err)
	err = s.network.App.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, originAccAddr, coins)
	s.Require().NoError(err)

	// Grant MintHaqqAuthorization from origin to dummyContract with a spend limit
	spendCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewIntFromBigInt(burnAmount).MulRaw(2))
	mintAuthz := &ethiqtypes.MintHaqqAuthorization{SpendLimit: &spendCoin}
	expiration := ctx.BlockTime().Add(time.Hour).UTC()
	err = s.network.App.AuthzKeeper.SaveGrant(ctx, dummyContract.Bytes(), origin.Bytes(), mintAuthz, &expiration)
	s.Require().NoError(err)

	// Verify the grant was saved
	savedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, dummyContract.Bytes(), origin.Bytes(), ethiq.MintHaqqMsgURL)
	s.Require().NotNil(savedAuthz)

	// Create contract where CallerAddress = dummyContract (not origin)
	// This simulates a call through a proxy/dummy smart contract
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, dummyContract, s.precompile, 200000)

	method := s.precompile.Methods[ethiq.MintHaqq]
	bz, err := s.precompile.MintHaqq(
		ctx,
		origin,
		contract,
		s.network.GetStateDB(),
		&method,
		[]any{
			origin,   // sender == origin
			receiver, // receiver
			burnAmount,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(bz)

	// Verify return value
	result := new(big.Int).SetBytes(bz)
	s.Require().True(result.Sign() > 0, "expected positive haqqAmount in return value")
}

// ---------------------------------------------------------------------------
// TestMintHaqqViaContractWithoutGrant verifies that calling MintHaqq
// through a third-party contract without any authorization grant fails.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestMintHaqqViaContractWithoutGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	origin := s.keyring.GetAddr(0)
	receiver := s.keyring.GetAddr(1)
	dummyContract := utiltx.GenerateAddress()

	// No grant is set up – just attempt the call
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, dummyContract, s.precompile, 200000)
	method := s.precompile.Methods[ethiq.MintHaqq]

	_, err := s.precompile.MintHaqq(ctx, origin, contract, s.network.GetStateDB(), &method, []any{
		origin,
		receiver,
		big.NewInt(1e18),
	})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "authorization")
}

// ---------------------------------------------------------------------------
// TestMintHaqqViaContractUnlimitedGrant verifies that an unlimited grant
// (nil SpendLimit) allows the third-party contract to mint multiple times
// without being deleted.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestMintHaqqViaContractUnlimitedGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	origin := s.keyring.GetAddr(0)
	receiver := s.keyring.GetAddr(1)
	dummyContract := utiltx.GenerateAddress()
	burnAmount := big.NewInt(1e18)

	// Fund origin
	originAccAddr := sdk.AccAddress(origin.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewIntFromBigInt(burnAmount).MulRaw(4)))
	err := s.network.App.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins)
	s.Require().NoError(err)
	err = s.network.App.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, originAccAddr, coins)
	s.Require().NoError(err)

	// Create unlimited grant (SpendLimit == nil)
	mintAuthz := &ethiqtypes.MintHaqqAuthorization{SpendLimit: nil}
	expiration := ctx.BlockTime().Add(time.Hour).UTC()
	err = s.network.App.AuthzKeeper.SaveGrant(ctx, dummyContract.Bytes(), origin.Bytes(), mintAuthz, &expiration)
	s.Require().NoError(err)

	method := s.precompile.Methods[ethiq.MintHaqq]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, dummyContract, s.precompile, 200000)

	// First mint call
	bz, err := s.precompile.MintHaqq(ctx, origin, contract, s.network.GetStateDB(), &method, []any{
		origin, receiver, burnAmount,
	})
	s.Require().NoError(err)
	s.Require().NotNil(bz)

	// Grant should still exist (unlimited → Updated, not Deleted)
	savedAuthz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, dummyContract.Bytes(), origin.Bytes(), ethiq.MintHaqqMsgURL)
	s.Require().NotNil(savedAuthz)
}

// ---------------------------------------------------------------------------
// TestCheckAndAcceptAuthorizationIfNeeded covers types.go directly.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestCheckAndAcceptAuthorizationCallerIsOrigin() {
	s.SetupTest()
	ctx := s.network.GetContext()

	origin := s.keyring.GetAddr(0)

	// When CallerAddress == origin, no authz check needed → always succeeds
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, origin, s.precompile, 200000)
	msg := &ethiqtypes.MsgMintHaqq{
		FromAddress: sdk.AccAddress(origin.Bytes()).String(),
		ToAddress:   sdk.AccAddress(s.keyring.GetAddr(1).Bytes()).String(),
		IslmAmount:  sdkmath.NewInt(1e18),
	}

	err := ethiq.CheckAndAcceptAuthorizationIfNeeded(ctx, contract, origin, s.network.App.AuthzKeeper, msg)
	s.Require().NoError(err)
}

func (s *PrecompileTestSuite) TestCheckAndAcceptAuthorizationWrongType() {
	s.SetupTest()
	ctx := s.network.GetContext()

	origin := s.keyring.GetAddr(0)
	dummyContract := utiltx.GenerateAddress()

	// Save a grant with the wrong authorization type (use MintHaqqByApplicationIDAuthorization
	// as grantee for MsgMintHaqq message type — the type assertion will fail)
	wrongAuthz := &ethiqtypes.MintHaqqByApplicationIDAuthorization{
		ApplicationsList: []uint64{0},
	}
	expiration := ctx.BlockTime().Add(time.Hour).UTC()
	// Save under MintHaqqMsgURL so CheckAuthzExists finds it but type assertion fails
	err := s.network.App.AuthzKeeper.SaveGrant(ctx, dummyContract.Bytes(), origin.Bytes(), wrongAuthz, &expiration)
	s.Require().NoError(err)

	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, dummyContract, s.precompile, 200000)
	msg := &ethiqtypes.MsgMintHaqq{
		FromAddress: sdk.AccAddress(origin.Bytes()).String(),
		ToAddress:   sdk.AccAddress(s.keyring.GetAddr(1).Bytes()).String(),
		IslmAmount:  sdkmath.NewInt(1e18),
	}

	err = ethiq.CheckAndAcceptAuthorizationIfNeeded(ctx, contract, origin, s.network.App.AuthzKeeper, msg)
	// Expected error: wrong authz type saved under the message URL
	s.Require().Error(err)
}

// ---------------------------------------------------------------------------
// TestAllowanceWithGrant covers the Allowance query with an existing grant.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestAllowanceWithGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	granter := s.keyring.GetAddr(0)
	grantee := common.BytesToAddress(utiltx.GenerateAddress().Bytes())
	limitAmount := sdkmath.NewInt(5e18)
	coin := sdk.NewCoin(utils.BaseDenom, limitAmount)
	authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: &coin}
	expiration := ctx.BlockTime().Add(time.Hour).UTC()
	err := s.network.App.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), authzGrant, &expiration)
	s.Require().NoError(err)

	method := s.precompile.Methods["allowance"]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, granter, s.precompile, 200000)

	bz, err := s.precompile.Allowance(ctx, &method, contract, []any{
		grantee,
		granter,
		mintHaqqMsgURL, // defined in query_test.go
	})
	s.Require().NoError(err)
	s.Require().NotNil(bz)

	// Unpack and check allowance equals limitAmount
	results, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Len(results, 1)
	allowance, ok := results[0].(*big.Int)
	s.Require().True(ok)
	s.Require().Equal(limitAmount.BigInt(), allowance)
}

// ---------------------------------------------------------------------------
// TestAllowanceUnlimitedGrant covers the nil SpendLimit → MaxUint256 branch.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestAllowanceUnlimitedGrant() {
	s.SetupTest()
	ctx := s.network.GetContext()

	granter := s.keyring.GetAddr(0)
	grantee := common.BytesToAddress(utiltx.GenerateAddress().Bytes())
	authzGrant := &ethiqtypes.MintHaqqAuthorization{SpendLimit: nil} // unlimited
	expiration := ctx.BlockTime().Add(time.Hour).UTC()
	err := s.network.App.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), authzGrant, &expiration)
	s.Require().NoError(err)

	method := s.precompile.Methods["allowance"]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, granter, s.precompile, 200000)

	bz, err := s.precompile.Allowance(ctx, &method, contract, []any{
		grantee,
		granter,
		mintHaqqMsgURL,
	})
	s.Require().NoError(err)
	s.Require().NotNil(bz)

	results, err := method.Outputs.Unpack(bz)
	s.Require().NoError(err)
	s.Require().Len(results, 1)
	allowance, ok := results[0].(*big.Int)
	s.Require().True(ok)
	s.Require().Equal(0, abi.MaxUint256.Cmp(allowance))
}

// ---------------------------------------------------------------------------
// TestMintHaqqByApplicationWithFunds covers the success path of MintHaqqByApplication.
// Application 6 uses address haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z and
// requires burning 100 aISLM (100e18 aISLM).
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestMintHaqqByApplicationWithFunds() {
	s.SetupTest()
	ctx := s.network.GetContext()

	// Get EVM address for the application's from-address
	appAccAddr := sdk.MustAccAddressFromBech32(appID6FromAddrBech32)
	appEvmAddr := common.BytesToAddress(appAccAddr.Bytes())

	// Fund the application's address (application 6 burn amount = 100e18 aISLM)
	burnAmount := sdkmath.NewInt(1e18).MulRaw(100)
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, burnAmount))
	err := s.network.App.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins)
	s.Require().NoError(err)
	err = s.network.App.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, appAccAddr, coins)
	s.Require().NoError(err)

	method := s.precompile.Methods[ethiq.MintHaqqByApplication]

	// CallerAddress == sender == origin → no authz required
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, appEvmAddr, s.precompile, 200000)

	bz, err := s.precompile.MintHaqqByApplication(ctx, appEvmAddr, contract, s.network.GetStateDB(), &method, []any{
		appEvmAddr,
		big.NewInt(6),
	})
	s.Require().NoError(err)
	s.Require().NotNil(bz)

	result := new(big.Int).SetBytes(bz)
	s.Require().True(result.Sign() > 0, "expected positive haqqAmount")
}

// ---------------------------------------------------------------------------
// TestNewMintHaqqAuthorization covers the types.go helper.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestNewMintHaqqAuthorization() {
	owner := s.keyring.GetAddr(0)
	spender := s.keyring.GetAddr(1)

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
			"",
		},
		{
			"fail - invalid owner (non-address)",
			[]any{"not-an-address", spender, big.NewInt(1e18)},
			true,
			"invalid owner",
		},
		{
			"fail - invalid spender (non-address)",
			[]any{owner, "not-an-address", big.NewInt(1e18)},
			true,
			"invalid spender",
		},
		{
			"fail - nil amount",
			[]any{owner, spender, nil},
			true,
			"invalid amount",
		},
		{
			"success - valid args",
			[]any{owner, spender, big.NewInt(1e18)},
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			retOwner, retSpender, authz, err := ethiq.NewMintHaqqAuthorization(tc.args)
			if tc.expError {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().Equal(owner, retOwner)
				s.Require().Equal(spender, retSpender)
				s.Require().NotNil(authz)
				s.Require().NotNil(authz.SpendLimit)
			}
		})
	}
}
