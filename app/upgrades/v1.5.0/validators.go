package v150

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
)

func (r *RevestingUpgradeHandler) storeBondedValidatorsByPower() error {
	// Store validators and their bonded tokens for further processing
	validators := r.StakingKeeper.GetBondedValidatorsByPower(r.ctx)
	if len(validators) > 25 {
		validators = validators[:25]
	}

	if len(validators) == 0 {
		return errors.New("no bonded validators found")
	}

	for _, validator := range validators {
		op := validator.GetOperator()
		r.vals[op.String()] = validator.GetTokens()
	}

	return nil
}

func (r *RevestingUpgradeHandler) getWeakestValidator() (*stakingtypes.Validator, *sdk.ValAddress, error) {
	var weakest string
	for valAddr := range r.vals {
		if weakest == "" {
			weakest = valAddr
			continue
		}

		if r.vals[valAddr].LT(r.vals[weakest]) {
			weakest = valAddr
		}
	}

	if weakest == "" {
		// Should never happen
		return nil, nil, errors.New("failed to find weakest validator from list")
	}

	weakestValAddr, err := sdk.ValAddressFromBech32(weakest)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse validator address")
	}

	val, found := r.StakingKeeper.GetValidator(r.ctx, weakestValAddr)
	if !found {
		// Should never happen, but just in case
		return nil, nil, errors.Wrapf(stakingtypes.ErrNoValidatorFound, "validator %s does not exist", weakestValAddr.String())
	}

	return &val, &weakestValAddr, nil
}

func (r *RevestingUpgradeHandler) reduceValidatorPower(valAddr string, amount math.Int) {
	power, ok := r.vals[valAddr]
	if !ok {
		return
	}

	r.vals[valAddr] = power.Sub(amount)
}

func (r *RevestingUpgradeHandler) increaseValidatorPower(valAddr string, amount math.Int) {
	power, ok := r.vals[valAddr]
	if !ok {
		return
	}

	r.vals[valAddr] = power.Add(amount)
}
