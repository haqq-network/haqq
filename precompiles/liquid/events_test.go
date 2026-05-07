package liquid_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/liquid"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	liquidtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
)

// maxUint256 returns 2^256 - 1, the largest value representable by a uint256.
func maxUint256() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

// TestEmitLiquidateEvent verifies that the Liquidate event is emitted with
// the expected address, signature topic, block number and ABI-encoded payload.
//
// Note: zero-amount is not covered here on purpose. MsgLiquidate.ValidateBasic
// rejects zero coins before EmitLiquidateEvent is ever reached, so the path is
// unreachable in production.
func (s *PrecompileTestSuite) TestEmitLiquidateEvent() {
	testcases := []struct {
		name          string
		sender        common.Address
		receiver      common.Address
		erc20Contract common.Address
		amount        *big.Int
	}{
		{
			name:          "pass",
			sender:        utiltx.GenerateAddress(),
			receiver:      utiltx.GenerateAddress(),
			erc20Contract: utiltx.GenerateAddress(),
			amount:        big.NewInt(1_000_000),
		},
		{
			// Edge case: full uint256 range. Confirms ABI encoding/decoding
			// round-trips the boundary value without truncation or sign issues.
			name:          "pass - max uint256 amount",
			sender:        utiltx.GenerateAddress(),
			receiver:      utiltx.GenerateAddress(),
			erc20Contract: utiltx.GenerateAddress(),
			amount:        maxUint256(),
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx := s.network.GetContext()
			stateDB := s.network.GetStateDB()

			err := s.precompile.EmitLiquidateEvent(
				ctx, stateDB, tc.sender, tc.receiver, tc.erc20Contract, tc.amount,
			)
			s.Require().NoError(err, "expected liquidate event to be emitted successfully")

			logs := stateDB.Logs()
			s.Require().Len(logs, 1, "expected exactly one log to be emitted")
			log := logs[0]

			s.Require().Equal(s.precompile.Address(), log.Address)

			event := s.precompile.ABI.Events[liquid.EventTypeLiquidate]
			s.Require().Equal(
				crypto.Keccak256Hash([]byte(event.Sig)),
				common.HexToHash(log.Topics[0].Hex()),
				"expected event signature to match",
			)
			//nolint: gosec // G115 blockHeight is positive int64 and can't overflow uint64
			s.Require().Equal(uint64(ctx.BlockHeight()), log.BlockNumber)

			// The Liquidate event has 2 indexed args (sender, receiver), so we
			// expect 1 (signature) + 2 (indexed) = 3 topics.
			s.Require().Len(log.Topics, 3, "expected 3 topics")

			var liquidateEvent liquid.EventLiquidate
			err = cmn.UnpackLog(s.precompile.ABI, &liquidateEvent, liquid.EventTypeLiquidate, *log)
			s.Require().NoError(err, "unable to unpack log into liquidate event")

			s.Require().Equal(tc.sender, liquidateEvent.Sender, "expected different sender address")
			s.Require().Equal(tc.receiver, liquidateEvent.Receiver, "expected different receiver address")
			s.Require().Equal(tc.amount, liquidateEvent.Amount, "expected different amount")
			s.Require().Equal(tc.erc20Contract, liquidateEvent.Erc20Contract, "expected different erc20 contract address")
		})
	}
}

// TestEmitRedeemEvent verifies that the Redeem event is emitted with the
// expected address, signature topic, block number and ABI-encoded payload.
func (s *PrecompileTestSuite) TestEmitRedeemEvent() {
	testcases := []struct {
		name     string
		sender   common.Address
		receiver common.Address
		denom    string
		amount   *big.Int
	}{
		{
			name:     "pass",
			sender:   utiltx.GenerateAddress(),
			receiver: utiltx.GenerateAddress(),
			denom:    liquidtypes.DenomBaseNameFromID(0),
			amount:   big.NewInt(1_000_000),
		},
		{
			name:     "pass - long denom id",
			sender:   utiltx.GenerateAddress(),
			receiver: utiltx.GenerateAddress(),
			denom:    liquidtypes.DenomBaseNameFromID(42),
			amount:   big.NewInt(123_456_789),
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx := s.network.GetContext()
			stateDB := s.network.GetStateDB()

			err := s.precompile.EmitRedeemEvent(
				ctx, stateDB, tc.sender, tc.receiver, tc.denom, tc.amount,
			)
			s.Require().NoError(err, "expected redeem event to be emitted successfully")

			logs := stateDB.Logs()
			s.Require().Len(logs, 1, "expected exactly one log to be emitted")
			log := logs[0]

			s.Require().Equal(s.precompile.Address(), log.Address)

			event := s.precompile.ABI.Events[liquid.EventTypeRedeem]
			s.Require().Equal(
				crypto.Keccak256Hash([]byte(event.Sig)),
				common.HexToHash(log.Topics[0].Hex()),
				"expected event signature to match",
			)
			//nolint: gosec // G115 blockHeight is positive int64 and can't overflow uint64
			s.Require().Equal(uint64(ctx.BlockHeight()), log.BlockNumber)

			// The Redeem event has 2 indexed args (sender, receiver), so we
			// expect 1 (signature) + 2 (indexed) = 3 topics.
			s.Require().Len(log.Topics, 3, "expected 3 topics")

			var redeemEvent liquid.EventRedeem
			err = cmn.UnpackLog(s.precompile.ABI, &redeemEvent, liquid.EventTypeRedeem, *log)
			s.Require().NoError(err, "unable to unpack log into redeem event")

			s.Require().Equal(tc.sender, redeemEvent.Sender, "expected different sender address")
			s.Require().Equal(tc.receiver, redeemEvent.Receiver, "expected different receiver address")
			s.Require().Equal(tc.denom, redeemEvent.Denom, "expected different denom")
			s.Require().Equal(tc.amount, redeemEvent.Amount, "expected different amount")
		})
	}
}
