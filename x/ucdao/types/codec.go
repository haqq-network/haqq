package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/x/authz"

	cryptocodec "github.com/haqq-network/haqq/crypto/codec"
)

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the dao module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgFund{}, "haqq/ucdao/MsgFund", nil)
	cdc.RegisterConcrete(&MsgFundLegacy{}, "haqq/dao/MsgFund", nil)
	cdc.RegisterConcrete(&MsgTransferOwnership{}, "haqq/ucdao/MsgTransferOwnership", nil)
	cdc.RegisterConcrete(&MsgTransferOwnershipWithRatio{}, "haqq/ucdao/MsgTransferWithRatio", nil)
	cdc.RegisterConcrete(&MsgTransferOwnershipWithAmount{}, "haqq/ucdao/MsgTransferWithAmount", nil)

	cdc.RegisterConcrete(&ConvertToHaqqAuthorization{}, "haqq/ucdao/ConvertToHaqqAuthorization", nil)
	cdc.RegisterConcrete(&TransferOwnershipAuthorization{}, "haqq/ucdao/TransferOwnershipAuthorization", nil)

	cdc.RegisterConcrete(Params{}, "haqq/x/ucdao/Params", nil)
}

// RegisterInterfaces registers the interfaces types with the Interface Registry.
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgFund{},
		&MsgTransferOwnership{},
		&MsgTransferOwnershipWithRatio{},
		&MsgTransferOwnershipWithAmount{},
	)

	registry.RegisterImplementations(
		(*authz.Authorization)(nil),
		&ConvertToHaqqAuthorization{},
		&TransferOwnershipAuthorization{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)
}
