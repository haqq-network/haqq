package staking_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/staking"
)

// TestAllowanceMaxUint256DoesNotPanic pins the fix for the nil-coin dereference in
// increaseAllowance / decreaseAllowance.
//
// authorization.CheckApprovalArgs encodes "unlimited" as a nil *sdk.Coin for the
// MaxUint256 argument. Before the fix both helpers checked only whether the stored
// limit was nil and then dereferenced coin.Amount, so a finite grant followed by
// increaseAllowance(grantee, type(uint256).max, ...) panicked out of the precompile:
// cmn.HandleGasError recovers ErrorOutOfGas only and re-raises everything else.
//
// The expected behaviour is a plain error. The grant must survive untouched.
func (s *PrecompileTestSuite) TestAllowanceMaxUint256DoesNotPanic() {
	maxUint256 := new(big.Int).Set(abi.MaxUint256)

	for _, tc := range []struct {
		name   string
		method string
		call   func(method abi.Method, args []interface{}) ([]byte, error)
	}{
		{
			name:   "increaseAllowance",
			method: authorization.IncreaseAllowanceMethod,
		},
		{
			name:   "decreaseAllowance",
			method: authorization.DecreaseAllowanceMethod,
		},
	} {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx := s.network.GetContext()
			stDB := s.network.GetStateDB()

			granter := s.keyring.GetKey(0)
			grantee := s.keyring.GetKey(1)

			// A finite grant: this is what makes the nil coin reachable, because both
			// helpers return early when the stored limit is nil.
			approveMethod := s.precompile.Methods[authorization.ApproveMethod]
			s.ApproveAndCheckAuthz(approveMethod, granter, grantee, staking.DelegateMsg, big.NewInt(1e18))

			method := s.precompile.Methods[tc.method]
			args := []interface{}{grantee.Addr, maxUint256, []string{staking.DelegateMsg}}

			var (
				bz  []byte
				err error
			)
			s.Require().NotPanics(func() {
				if tc.method == authorization.IncreaseAllowanceMethod {
					bz, err = s.precompile.IncreaseAllowance(ctx, granter.Addr, stDB, &method, args)
				} else {
					bz, err = s.precompile.DecreaseAllowance(ctx, granter.Addr, stDB, &method, args)
				}
			}, "MaxUint256 must not panic out of the precompile")

			s.Require().Error(err)
			s.Require().ErrorContains(err, "does not support unlimited amount")
			s.Require().Empty(bz)

			// The grant is untouched.
			authzGrant, _ := CheckAuthorizationWithContext(ctx, s.network.App.AuthzKeeper, staking.DelegateAuthz, grantee.Addr, granter.Addr)
			s.Require().NotNil(authzGrant)
			s.Require().NotNil(authzGrant.MaxTokens)
			s.Require().Equal(big.NewInt(1e18), authzGrant.MaxTokens.Amount.BigInt())
		})
	}
}

// TestIncreaseAllowanceOverflowIsRejected pins the companion guard: both operands are
// uint256 ABI arguments, so their sum can need 257 bits and sdkmath.Int.Add panics
// above sdkmath.MaxBitLen.
func (s *PrecompileTestSuite) TestIncreaseAllowanceOverflowIsRejected() {
	s.SetupTest()
	ctx := s.network.GetContext()
	stDB := s.network.GetStateDB()

	granter := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)

	// Largest limit that still fits: 2^256 - 1 would be the unlimited sentinel, so use
	// one below it and then add a value that pushes the sum over 256 bits.
	nearMax := new(big.Int).Sub(abi.MaxUint256, big.NewInt(1))

	approveMethod := s.precompile.Methods[authorization.ApproveMethod]
	s.ApproveAndCheckAuthz(approveMethod, granter, grantee, staking.DelegateMsg, nearMax)

	method := s.precompile.Methods[authorization.IncreaseAllowanceMethod]
	args := []interface{}{grantee.Addr, big.NewInt(1e18), []string{staking.DelegateMsg}}

	var err error
	s.Require().NotPanics(func() {
		_, err = s.precompile.IncreaseAllowance(ctx, granter.Addr, stDB, &method, args)
	}, "an out-of-range allowance must not panic out of the precompile")
	s.Require().Error(err)
	s.Require().ErrorContains(err, "does not fit in")
}
