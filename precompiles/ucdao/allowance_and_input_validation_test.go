package ucdao_test

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	"github.com/haqq-network/haqq/utils"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestAllowanceMaxUint256DoesNotPanic is the ucDAO half of the staking regression with
// the same name: authorization.CheckApprovalArgs encodes "unlimited" as a nil *sdk.Coin
// for MaxUint256, and both allowance helpers used to dereference it once the stored
// limit was non-nil. A panic here unwinds out of the precompile, because
// cmn.HandleGasError recovers ErrorOutOfGas only.
func (s *PrecompileTestSuite) TestAllowanceMaxUint256DoesNotPanic() {
	maxUint256 := new(big.Int).Set(abi.MaxUint256)

	for _, msgURL := range []string{ucdao.ConvertToHaqqMsgURL, ucdao.TransferOwnershipWithAmountMsgURL} {
		for _, methodName := range []string{
			authorization.IncreaseAllowanceMethod,
			authorization.DecreaseAllowanceMethod,
		} {
			s.Run(methodName+" "+msgURL, func() {
				s.SetupTest()
				ctx := s.network.GetContext()
				stDB := s.network.GetStateDB()

				granter := s.keyring.GetKey(0)
				grantee := s.keyring.GetKey(1)

				// A finite grant: this is what makes the nil coin reachable, since both
				// helpers return early when the stored SpendLimit is nil.
				approveMethod := s.precompile.Methods[authorization.ApproveMethod]
				_, err := s.precompile.Approve(ctx, granter.Addr, stDB, &approveMethod, []interface{}{
					grantee.Addr, big.NewInt(1e18), []string{msgURL},
				})
				s.Require().NoError(err)

				method := s.precompile.Methods[methodName]
				args := []interface{}{grantee.Addr, maxUint256, []string{msgURL}}

				var bz []byte
				s.Require().NotPanics(func() {
					if methodName == authorization.IncreaseAllowanceMethod {
						bz, err = s.precompile.IncreaseAllowance(ctx, granter.Addr, stDB, &method, args)
					} else {
						bz, err = s.precompile.DecreaseAllowance(ctx, granter.Addr, stDB, &method, args)
					}
				}, "MaxUint256 must not panic out of the precompile")

				s.Require().Error(err)
				s.Require().ErrorContains(err, "does not support unlimited amount")
				s.Require().Empty(bz)

				// The grant is untouched, and readable under the URL it was approved
				// with - see TestTransferOwnershipGrantRoundTrip.
				grant, _ := s.network.App.AuthzKeeper.GetAuthorization(
					ctx, grantee.AccAddr, granter.AccAddr, msgURL,
				)
				s.Require().NotNil(grant)
			})
		}
	}
}

// TestTransferOwnershipWithAmountRejectsBadInput pins the fix for the panic on
// attacker-controlled denoms: transferOwnershipWithAmount takes a string[] straight
// from calldata and used to feed it to sdk.NewCoin, which panics on a malformed denom.
// Every case below must be an error, never a panic.
func (s *PrecompileTestSuite) TestTransferOwnershipWithAmountRejectsBadInput() {
	owner := common.HexToAddress("0x1000000000000000000000000000000000000001")
	newOwner := common.HexToAddress("0x1000000000000000000000000000000000000002")

	testCases := []struct {
		name        string
		denoms      []string
		amounts     []*big.Int
		errContains string
	}{
		{"empty denom", []string{""}, []*big.Int{big.NewInt(1)}, "invalid coin at index 0"},
		{"malformed denom", []string{"!!bad"}, []*big.Int{big.NewInt(1)}, "invalid coin at index 0"},
		{"denom with space", []string{"a ISLM"}, []*big.Int{big.NewInt(1)}, "invalid coin at index 0"},
		{
			"overlong denom",
			[]string{strings.Repeat("a", 200)},
			[]*big.Int{big.NewInt(1)},
			"invalid coin at index 0",
		},
		{
			"duplicate denom",
			[]string{utils.BaseDenom, utils.BaseDenom},
			[]*big.Int{big.NewInt(1), big.NewInt(1)},
			"duplicate denom",
		},
		{
			"amount wider than 256 bits",
			[]string{utils.BaseDenom},
			[]*big.Int{new(big.Int).Lsh(big.NewInt(1), 256)},
			"does not fit in",
		},
		{"nil amount", []string{utils.BaseDenom}, []*big.Int{nil}, "nil amount at index 0"},
		{
			"length mismatch",
			[]string{utils.BaseDenom},
			[]*big.Int{big.NewInt(1), big.NewInt(2)},
			"length mismatch",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			args := []interface{}{owner, newOwner, tc.denoms, tc.amounts}

			var err error
			s.Require().NotPanics(func() {
				_, _, _, err = ucdao.NewTransferOwnershipWithAmountMsg(args)
			}, "malformed calldata must not panic out of the precompile")
			s.Require().Error(err)
			s.Require().ErrorContains(err, tc.errContains)
		})
	}

	// Sanity: a well-formed call still builds a valid message.
	msg, _, _, err := ucdao.NewTransferOwnershipWithAmountMsg([]interface{}{
		owner, newOwner, []string{utils.BaseDenom}, []*big.Int{big.NewInt(1e18)},
	})
	s.Require().NoError(err)
	s.Require().NoError(msg.ValidateBasic())
	s.Require().Equal(int64(1e18), msg.Amount.AmountOf(utils.BaseDenom).Int64())
}

// TestConvertToHaqqMsgIsValidated pins the ValidateBasic call added to the message
// constructors: the precompile calls the msg server directly, so baseapp's
// validateBasicTxMsgs never runs on this path.
func (s *PrecompileTestSuite) TestConvertToHaqqMsgIsValidated() {
	sender := common.HexToAddress("0x1000000000000000000000000000000000000001")
	receiver := common.HexToAddress("0x1000000000000000000000000000000000000002")

	_, _, _, err := ucdao.NewConvertToHaqqMsg([]interface{}{sender, receiver, big.NewInt(0)})
	s.Require().Error(err, "a zero islmAmount must be rejected before the msg server")
	s.Require().ErrorContains(err, "islmAmount must be positive")

	msg, _, _, err := ucdao.NewConvertToHaqqMsg([]interface{}{sender, receiver, big.NewInt(1e18)})
	s.Require().NoError(err)
	s.Require().Equal(ucdaotypes.TypeMsgConvertToHaqq, msg.Type())
}
