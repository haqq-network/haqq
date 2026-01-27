package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type FundSource int8

const (
	FundSource_BANK  FundSource = 1
	FundSource_UCDAO FundSource = 2
)

var (
	FundSources = map[FundSource]string{
		FundSource_BANK:  "bank",
		FundSource_UCDAO: "ucdao",
	}
)

type Application struct {
	FromAddress string
	ToAddress   string
	FundSource  FundSource
	IslmAmount  string
}

func (a Application) ValidateAndParse() (sdk.AccAddress, sdk.AccAddress, sdkmath.Int, error) {
	toAddress, err := sdk.AccAddressFromBech32(a.ToAddress)
	if err != nil {
		return nil, nil, sdkmath.ZeroInt(), errorsmod.Wrapf(ErrInvalidAddress, "invalid to_address: %v", err)
	}

	fromAddress, err := sdk.AccAddressFromBech32(a.FromAddress)
	if err != nil {
		return nil, nil, sdkmath.ZeroInt(), errorsmod.Wrapf(ErrInvalidAddress, "invalid from_address: %v", err)
	}

	if a.FundSource != FundSource_BANK && a.FundSource != FundSource_UCDAO {
		return nil, nil, sdkmath.ZeroInt(), errorsmod.Wrap(ErrInvalidFundsSource, "invalid fund_source")
	}

	islmAmount, ok := sdkmath.NewIntFromString(a.IslmAmount)
	if !ok {
		return nil, nil, sdkmath.ZeroInt(), fmt.Errorf("invalid islm_amount: %s", a.IslmAmount)
	}

	if islmAmount.LTE(sdkmath.ZeroInt()) {
		return nil, nil, sdkmath.ZeroInt(), errorsmod.Wrap(ErrInvalidAmount, "islm_amount must be positive and greater than zero")
	}

	return fromAddress, toAddress, islmAmount, nil
}
