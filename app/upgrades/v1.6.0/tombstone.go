package v160

import (
	"fmt"
	"sort"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var validators = map[string]string{
	"haqqvaloper1p02zk5ecdanap637e2wtt82cucjlxtkrhus623": "5704902600000000000000000",
	"haqqvaloper1xp597fjhgu6dx3a525htulkn36fqqntjaqvhct": "400003650000000000000000",
	"haqqvaloper1xwpeshg6v7fdg55evd9pzlay22a407dnxaat00": "400007650000000000000000",
	"haqqvaloper1s8p7kahfwghxz56elmdufgrvehzx20a99q9v7n": "406662350000000000000000",
	"haqqvaloper1hgggrfgjeu4d5nveh03c6w37magsuqcy84p44t": "6250096250000000000000000",
}

// RevertTombstone attempts to revert a tombstone state of a validator.
func RevertTombstone(
	ctx sdk.Context,
	stk stakingkeeper.Keeper,
	slk slashingkeeper.Keeper,
	bk bankkeeper.Keeper,
) error {
	logger := ctx.Logger()
	logger.Info("Revert tombstoning...")

	keys := make([]string, 0, len(validators))
	for k := range validators {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		valAddr, err := sdk.ValAddressFromBech32(k)
		if err != nil {
			return errors.Wrapf(err, "validator address is not valid bech32: %s", k)
		}

		val := stk.Validator(ctx, valAddr)
		if val == nil {
			return errors.Wrapf(err, "validator not found: %s", k)
		}

		valStruct := val.(stakingtypes.Validator)

		consAddr, err := valStruct.GetConsAddr()
		if err != nil {
			return errors.Wrapf(err, "get validator's consensus address: %s", k)
		}

		signInfo, ok := slk.GetValidatorSigningInfo(ctx, consAddr)
		if !ok {
			return errors.Wrapf(err, "signing info not found: %s", k)
		}

		if !signInfo.Tombstoned {
			return errors.Wrapf(err, "validator is not tombstoned: %s", k)
		}

		// Revert tombstone info
		signInfo.Tombstoned = false
		slk.SetValidatorSigningInfo(ctx, consAddr, signInfo)

		// Set jail until=now, the validator then must unjail manually
		slk.JailUntil(ctx, consAddr, ctx.BlockTime())

		tokens, ok := math.NewIntFromString(validators[k])
		if !ok {
			return errors.Wrapf(err, "parse tokens for validator: %s", k)
		}

		// Refund
		coinsToRefund := sdk.NewCoins(sdk.NewCoin(stk.BondDenom(ctx), tokens))
		poolAddr := stakingtypes.BondedPoolName
		if !valStruct.IsBonded() {
			poolAddr = stakingtypes.NotBondedPoolName
		}
		if err := bk.SendCoinsFromModuleToModule(ctx, distrtypes.ModuleName, poolAddr, coinsToRefund); err != nil {
			return errors.Wrapf(err, "refund tokens for validator: %s - %s", k, coinsToRefund.String())
		}

		valStruct.Tokens = valStruct.Tokens.Add(tokens)
		stk.SetValidator(ctx, valStruct)

		logger.Info(fmt.Sprintf("Revived validator - %s: refund slashed amount %s", k, coinsToRefund.String()))
	}

	return nil
}
