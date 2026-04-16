package types

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/haqq-network/haqq/utils"
)

func NewConvertToHaqqAuthorization(amount *sdk.Coin) (*ConvertToHaqqAuthorization, error) {
	a := &ConvertToHaqqAuthorization{
		SpendLimit: amount,
	}

	if err := a.ValidateBasic(); err != nil {
		return nil, err
	}

	return a, nil
}

func (m *ConvertToHaqqAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgConvertToHaqq{})
}

func (m *ConvertToHaqqAuthorization) ValidateBasic() error {
	if m.SpendLimit != nil {
		if err := m.SpendLimit.Validate(); err != nil {
			return err
		}
		if m.SpendLimit.Denom != utils.BaseDenom {
			return fmt.Errorf("spend limit denom must be %s, got %s", utils.BaseDenom, m.SpendLimit.Denom)
		}
		if !m.SpendLimit.Amount.IsPositive() {
			return fmt.Errorf("spend limit amount must be positive")
		}
	}

	return nil
}

func (m *ConvertToHaqqAuthorization) Accept(_ context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	convertMsg, ok := msg.(*MsgConvertToHaqq)
	if !ok {
		return authz.AcceptResponse{}, fmt.Errorf("expected %T, got %T", &MsgConvertToHaqq{}, msg)
	}

	if m.SpendLimit != nil {
		if convertMsg.IslmAmount.GT(m.SpendLimit.Amount) {
			return authz.AcceptResponse{Accept: false}, fmt.Errorf("requested amount %s exceeds spend limit %s", convertMsg.IslmAmount, m.SpendLimit.Amount)
		}

		// Update spend limit by subtracting the amount
		remaining := m.SpendLimit.Amount.Sub(convertMsg.IslmAmount)
		if remaining.IsZero() {
			// Delete the authorization if limit is exhausted
			return authz.AcceptResponse{
				Accept: true,
				Delete: true,
			}, nil
		}

		// Update the authorization with remaining amount
		updatedAuthz := &ConvertToHaqqAuthorization{
			SpendLimit: &sdk.Coin{
				Denom:  m.SpendLimit.Denom,
				Amount: remaining,
			},
		}

		return authz.AcceptResponse{
			Accept:  true,
			Updated: updatedAuthz,
			Delete:  false,
		}, nil
	}

	// No spend limit, accept without updating
	return authz.AcceptResponse{Accept: true}, nil
}

func NewTransferOwnershipAuthorization(amount *sdk.Coin) (*TransferOwnershipAuthorization, error) {
	a := &TransferOwnershipAuthorization{
		SpendLimit: amount,
	}

	if err := a.ValidateBasic(); err != nil {
		return nil, err
	}

	return a, nil
}

func (m *TransferOwnershipAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgTransferOwnershipWithAmount{})
}

func (m *TransferOwnershipAuthorization) ValidateBasic() error {
	if m.SpendLimit != nil {
		if err := m.SpendLimit.Validate(); err != nil {
			return err
		}
		if m.SpendLimit.Denom != utils.BaseDenom {
			return fmt.Errorf("spend limit denom must be %s, got %s", utils.BaseDenom, m.SpendLimit.Denom)
		}
		if !m.SpendLimit.Amount.IsPositive() {
			return fmt.Errorf("spend limit amount must be positive")
		}
	}

	return nil
}

func (m *TransferOwnershipAuthorization) Accept(_ context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	toMsg, ok := msg.(*MsgTransferOwnershipWithAmount)
	if !ok {
		return authz.AcceptResponse{}, fmt.Errorf("expected %T, got %T", &MsgTransferOwnershipWithAmount{}, msg)
	}

	if toMsg.Amount.IsZero() {
		return authz.AcceptResponse{}, fmt.Errorf("requested amount should be greater than zero, got %s", toMsg.Amount.String())
	}

	if err := toMsg.Amount.Validate(); err != nil {
		return authz.AcceptResponse{}, fmt.Errorf("invalid amount %s: %w", toMsg.Amount.String(), err)
	}

	totalTransferAmount := sdkmath.ZeroInt()
	for _, coin := range toMsg.Amount {
		if coin.Denom != utils.BaseDenom && !utils.IsLiquidToken(coin.Denom) {
			// Maybe we should throw an error here
			continue
		}

		totalTransferAmount = totalTransferAmount.Add(coin.Amount)
	}

	if m.SpendLimit != nil {
		if totalTransferAmount.GT(m.SpendLimit.Amount) {
			return authz.AcceptResponse{Accept: false}, fmt.Errorf("requested amount %s exceeds spend limit %s", toMsg.Amount.String(), m.SpendLimit.Amount)
		}

		// Update spend limit by subtracting the amount
		remaining := m.SpendLimit.Amount.Sub(totalTransferAmount)
		if remaining.IsZero() {
			// Delete the authorization if limit is exhausted
			return authz.AcceptResponse{
				Accept: true,
				Delete: true,
			}, nil
		}

		// Update the authorization with remaining amount
		updatedAuthz := &TransferOwnershipAuthorization{
			SpendLimit: &sdk.Coin{
				Denom:  m.SpendLimit.Denom,
				Amount: remaining,
			},
		}

		return authz.AcceptResponse{
			Accept:  true,
			Updated: updatedAuthz,
			Delete:  false,
		}, nil
	}

	// No spend limit, accept without updating
	return authz.AcceptResponse{Accept: true}, nil
}
