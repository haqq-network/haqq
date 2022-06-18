package app

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// func (k Keeper) AfterValidatorBonded(ctx sdk.Context, address sdk.ConsAddress, _ sdk.ValAddress) {
// 	// Update the signing info start height or create a new signing info
// 	_, found := k.GetValidatorSigningInfo(ctx, address)
// 	if !found {
// 		signingInfo := types.NewValidatorSigningInfo(
// 			address,
// 			ctx.BlockHeight(),
// 			0,
// 			time.Unix(0, 0),
// 			false,
// 			0,
// 		)
// 		k.SetValidatorSigningInfo(ctx, address, signingInfo)
// 	}
// }

// AfterValidatorCreated adds the address-pubkey relation when a validator is created.
// func (k Keeper) AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress) error {
// 	validator := k.sk.Validator(ctx, valAddr)
// 	consPk, err := validator.ConsPubKey()
// 	if err != nil {
// 		return err
// 	}
// 	k.AddPubkey(ctx, consPk)

// 	return nil
// }

// AfterValidatorRemoved deletes the address-pubkey relation when a validator is removed,
// func (k Keeper) AfterValidatorRemoved(ctx sdk.Context, address sdk.ConsAddress) {
// 	k.deleteAddrPubkeyRelation(ctx, crypto.Address(address))
// }

// Hooks wrapper struct for slashing keeper
type Hooks struct {
	// k Keeper
}

var _ types.StakingHooks = Hooks{}

// Return the wrapper struct
func HaqqStakingHooks() Hooks {
	return Hooks{}
}

// Implements sdk.ValidatorHooks
func (h Hooks) AfterValidatorBonded(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) {
	fmt.Println("HAQQ HOOK: AfterValidatorBonded!!!!!!")
	fmt.Println("HAQQ HOOK: AfterValidatorBonded!!!!!!")
	fmt.Println("HAQQ HOOK: AfterValidatorBonded!!!!!!")
}

// Implements sdk.ValidatorHooks
func (h Hooks) AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress) {
	// h.k.AfterValidatorCreated(ctx, valAddr)
	fmt.Println("HAQQ HOOK: AfterValidatorCreated!!!!!!")
	fmt.Println("HAQQ HOOK: AfterValidatorCreated!!!!!!")
	fmt.Println("HAQQ HOOK: AfterValidatorCreated!!!!!!")

	fmt.Println("AfterValidatorCreated ctx:", ctx)
	fmt.Println("AfterValidatorCreated valAddr:", valAddr)
}

func (h Hooks) AfterValidatorRemoved(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) {
}

func (h Hooks) AfterValidatorBeginUnbonding(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) {
}

func (h Hooks) BeforeValidatorModified(_ sdk.Context, _ sdk.ValAddress) {
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
}

func (h Hooks) BeforeDelegationCreated(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) {
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
}

func (h Hooks) BeforeDelegationSharesModified(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) {
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
}

func (h Hooks) BeforeDelegationRemoved(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) {
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorModified!!!!!!")
}

func (h Hooks) AfterDelegationModified(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) {
	fmt.Println("HAQQ HOOK: AfterDelegationModified!!!!!!")
	fmt.Println("HAQQ HOOK: AfterDelegationModified!!!!!!")
	fmt.Println("HAQQ HOOK: AfterDelegationModified!!!!!!")
}

func (h Hooks) BeforeValidatorSlashed(_ sdk.Context, _ sdk.ValAddress, _ sdk.Dec) {
	fmt.Println("HAQQ HOOK: BeforeValidatorSlashed!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorSlashed!!!!!!")
	fmt.Println("HAQQ HOOK: BeforeValidatorSlashed!!!!!!")
}
