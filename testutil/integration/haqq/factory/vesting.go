package factory

import (
	"fmt"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

type VestingTxFactory interface {
	// CreateClawbackVestingAccount is a method to create and broadcast a MsgCreateClawbackVestingAccount
	CreateClawbackVestingAccount(vestingPriv cryptotypes.PrivKey, funderAddr sdk.AccAddress, enableGovClawback bool) error
	// UpdateVestingFunder is a method to create and broadcast a MsgUpdateVestingFunder
	UpdateVestingFunder(funderpriv cryptotypes.PrivKey, newFunderAddr sdk.AccAddress, vestingAddr sdk.AccAddress) error
}

type vestingTxFactory struct {
	commonfactory.BaseTxFactory
}

func newVestingTxFactory(bf commonfactory.BaseTxFactory) VestingTxFactory {
	return &vestingTxFactory{bf}
}

// SetupClawbackVestingAccount sets up a clawback vesting account
// using the TestVestingSchedule. If exceeded balance is provided,
// will fund the vesting account with it.
func (tf *vestingTxFactory) SetupClawbackVestingAccount(funderPriv cryptotypes.PrivKey, vestingAddr sdk.AccAddress) error {
	//funderAccAddr := sdk.AccAddress(funderPriv.PubKey().Address())

	// TODO Implement!

	return nil
}

// CreateClawbackVestingAccount in the provided address, with the provided
// funder address
func (tf *vestingTxFactory) CreateClawbackVestingAccount(vestingPriv cryptotypes.PrivKey, funderAddr sdk.AccAddress, enableGovClawback bool) error {
	//vestingAccAddr := sdk.AccAddress(vestingPriv.PubKey().Address())
	//
	//msg := vestingtypes.NewMsgCreateClawbackVestingAccount(
	//	funderAddr,
	//	vestingAccAddr,
	//	enableGovClawback,
	//)
	//
	//resp, err := tf.ExecuteCosmosTx(vestingPriv, commonfactory.CosmosTxArgs{
	//	Msgs: []sdk.Msg{msg},
	//})
	//
	//if resp.Code != 0 {
	//	err = fmt.Errorf("received error code %d on CreateClawbackVestingAccount transaction. Logs: %s", resp.Code, resp.Log)
	//}
	//
	//return err

	// TODO fix
	return nil
}

// UpdateVestingFunder with the given new funder.
func (tf *vestingTxFactory) UpdateVestingFunder(funderPriv cryptotypes.PrivKey, newFunderAddr, vestingAddr sdk.AccAddress) error {
	funderAccAddr := sdk.AccAddress(funderPriv.PubKey().Address())

	msg := vestingtypes.NewMsgUpdateVestingFunder(
		funderAccAddr,
		newFunderAddr,
		vestingAddr,
	)

	resp, err := tf.ExecuteCosmosTx(funderPriv, commonfactory.CosmosTxArgs{
		Msgs: []sdk.Msg{msg},
	})

	if resp.Code != 0 {
		err = fmt.Errorf("received error code %d on UpdateVestingFunder transaction. Logs: %s", resp.Code, resp.Log)
	}

	return err
}
