package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/haqq-network/haqq/utils"
)

func (m *MintHaqqAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgMintHaqq{})
}

func (m *MintHaqqAuthorization) Accept(ctx context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
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

func (m *MintHaqqByApplicationIDAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgMintHaqqByApplication{})
}

func (m *MintHaqqByApplicationIDAuthorization) Accept(ctx context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	mintMsg, ok := msg.(*MsgMintHaqqByApplication)
	if !ok {
		return authz.AcceptResponse{}, fmt.Errorf("expected %T, got %T", &MsgMintHaqqByApplication{}, msg)
	}

	// Check if application ID is in allow list (if allow list is not empty)
	if len(m.ApplicationsList) > 0 {
		appID := mintMsg.ApplicationId
		allowed := false
		for _, allowedID := range m.ApplicationsList {
			if allowedID == appID {
				allowed = true
				break
			}
		}
		if !allowed {
			return authz.AcceptResponse{Accept: false}, fmt.Errorf("application ID %d is not in allow list", appID)
		}
	}

	// For application-based authorization, we delete after use since the whole application amount is spent
	return authz.AcceptResponse{
		Accept: true,
		Delete: true,
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
