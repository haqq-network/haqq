package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
)

type ApplicationListItem struct {
	ID                         uint64
	FromAddress                string
	ToAddress                  string
	FundSource                 SourceOfFunds
	IslmAmount                 string
	IslmAccumulatedBurntAmount string
}

func (a ApplicationListItem) AsBurnApplication() (*BurnApplication, error) {
	islmAmount, ok := sdkmath.NewIntFromString(a.IslmAmount)
	if !ok {
		return nil, fmt.Errorf("invalid islm_amount: %s", a.IslmAmount)
	}

	islmAccumulatedBurntAmount, ok := sdkmath.NewIntFromString(a.IslmAccumulatedBurntAmount)
	if !ok {
		return nil, fmt.Errorf("invalid islm_accumulated_burnt_amount: %s", a.IslmAccumulatedBurntAmount)
	}

	burnApplication := &BurnApplication{
		Id:                 a.ID,
		FromAddress:        a.FromAddress,
		ToAddress:          a.ToAddress,
		Source:             a.FundSource,
		BurnAmount:         sdk.NewCoin(utils.BaseDenom, islmAmount),
		BurnedBeforeAmount: sdk.NewCoin(utils.BaseDenom, islmAccumulatedBurntAmount),
		IsExecuted:         false,
		IsCanceled:         islmAmount.IsZero(),
	}

	if err := burnApplication.ValidateBasic(); err != nil {
		return nil, err
	}

	return burnApplication, nil
}

func (ba *BurnApplication) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(ba.FromAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid from_address: %v", err)
	}

	if _, err := sdk.AccAddressFromBech32(ba.ToAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid to_address: %v", err)
	}

	if ba.Source != SourceOfFunds_SOURCE_OF_FUNDS_BANK && ba.Source != SourceOfFunds_SOURCE_OF_FUNDS_UCDAO {
		return errorsmod.Wrapf(ErrInvalidFundsSource, "invalid source_of_funds: %d", ba.Source)
	}

	if !ba.BurnAmount.IsValid() {
		return errorsmod.Wrapf(ErrInvalidAmount, "islm_amount must be positive or zero; got %s", ba.BurnAmount.String())
	}

	if ba.BurnAmount.Denom != utils.BaseDenom {
		return errorsmod.Wrapf(ErrInvalidAmount, "invalid denom for islm_amount; expected aISLM, got %s", ba.BurnAmount.Denom)
	}

	if !ba.BurnedBeforeAmount.IsValid() {
		return errorsmod.Wrapf(ErrInvalidAmount, "islm_accumulated_burnt_amount must be positive or zero; got %s", ba.BurnedBeforeAmount.String())
	}

	if ba.BurnedBeforeAmount.Denom != utils.BaseDenom {
		return errorsmod.Wrapf(ErrInvalidAmount, "invalid denom for islm_accumulated_burnt_amount; expected aISLM, got %s", ba.BurnedBeforeAmount.Denom)
	}

	return nil
}
