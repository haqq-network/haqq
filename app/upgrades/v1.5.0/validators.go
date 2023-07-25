package v150

import (
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
)

func (r *RevestingUpgradeHandler) SetValidatorsList(valops []string) error {
	if len(valops) == 0 {
		return errors.New("no validators provided")
	}

	for _, valop := range valops {
		valAddr, err := sdk.ValAddressFromBech32(valop)
		if err != nil {
			return errors.Wrap(err, "failed to parse validator address")
		}

		val, found := r.StakingKeeper.GetValidator(r.ctx, valAddr)
		if !found {
			// Should never happen, but just in case
			return errors.Wrapf(stakingtypes.ErrNoValidatorFound, "validator %s does not exist", valAddr.String())
		}

		op := val.GetOperator()
		r.vals[op.String()] = val.GetTokens()
	}
	r.ctx.Logger().Info("Stored bonded validators before upgrade: " + strconv.Itoa(len(r.vals)))

	return nil
}

func (r *RevestingUpgradeHandler) GetValidatorsList() map[string]math.Int {
	return r.vals
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
