package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/haqq-network/haqq/utils"
)

func NewMintHaqqAuthorization(amount *sdk.Coin) (*MintHaqqAuthorization, error) {
	a := &MintHaqqAuthorization{
		SpendLimit: amount,
	}

	if err := a.ValidateBasic(); err != nil {
		return nil, err
	}

	return a, nil
}

func (m *MintHaqqAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgMintHaqq{})
}

func (m *MintHaqqAuthorization) Accept(_ context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	mintMsg, ok := msg.(*MsgMintHaqq)
	if !ok {
		return authz.AcceptResponse{}, fmt.Errorf("expected %T, got %T", &MsgMintHaqq{}, msg)
	}

	// Check spend limit
	if m.SpendLimit != nil {
		if mintMsg.IslmAmount.GT(m.SpendLimit.Amount) {
			return authz.AcceptResponse{Accept: false}, fmt.Errorf("requested amount %s exceeds spend limit %s", mintMsg.IslmAmount, m.SpendLimit.Amount)
		}

		// Update spend limit by subtracting the amount
		remaining := m.SpendLimit.Amount.Sub(mintMsg.IslmAmount)
		if remaining.IsZero() {
			// Delete the authorization if limit is exhausted
			return authz.AcceptResponse{
				Accept: true,
				Delete: true,
			}, nil
		}

		// Update the authorization with remaining amount
		updatedAuthz := &MintHaqqAuthorization{
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

func (m *MintHaqqAuthorization) ValidateBasic() error {
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

func NewMintHaqqByApplicationAuthorization(appID uint64) (*MintHaqqByApplicationIDAuthorization, error) {
	a := &MintHaqqByApplicationIDAuthorization{}
	a.ApplicationsList = make([]uint64, 0, 1)
	a.ApplicationsList = append(a.ApplicationsList, appID)

	if err := a.ValidateBasic(); err != nil {
		return nil, err
	}

	return a, nil
}

func (m *MintHaqqByApplicationIDAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgMintHaqqByApplication{})
}

func (m *MintHaqqByApplicationIDAuthorization) Accept(_ context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	mintMsg, ok := msg.(*MsgMintHaqqByApplication)
	if !ok {
		return authz.AcceptResponse{}, fmt.Errorf("expected %T, got %T", &MsgMintHaqqByApplication{}, msg)
	}

	appID := mintMsg.ApplicationId
	if len(m.ApplicationsList) == 0 {
		return authz.AcceptResponse{Accept: false}, fmt.Errorf("application ID %d is not in allow list", appID)
	}

	remaining := make([]uint64, 0, len(m.ApplicationsList))
	allowed := false
	for _, allowedID := range m.ApplicationsList {
		if allowedID == appID {
			allowed = true
			continue
		}
		remaining = append(remaining, allowedID)
	}
	if !allowed {
		return authz.AcceptResponse{Accept: false}, fmt.Errorf("application ID %d is not in allow list", appID)
	}

	if len(remaining) == 0 {
		return authz.AcceptResponse{
			Accept: true,
			Delete: true,
		}, nil
	}

	return authz.AcceptResponse{
		Accept: true,
		Updated: &MintHaqqByApplicationIDAuthorization{
			ApplicationsList: remaining,
		},
		Delete: false,
	}, nil
}

func (m *MintHaqqByApplicationIDAuthorization) ValidateBasic() error {
	// Validate applications list
	for _, appID := range m.ApplicationsList {
		if !IsApplicationExists(appID) {
			return ErrInvalidApplicationID
		}
	}

	return nil
}
