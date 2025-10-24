package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	cryptocodec "github.com/haqq-network/haqq/crypto/codec"
)

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the dao module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgFund{}, "haqq/ucdao/MsgFund")
	legacy.RegisterAminoMsg(cdc, &MsgFundLegacy{}, "haqq/dao/MsgFund")
	legacy.RegisterAminoMsg(cdc, &MsgTransferOwnership{}, "haqq/ucdao/MsgTransferOwnership")
	legacy.RegisterAminoMsg(cdc, &MsgTransferOwnershipWithRatio{}, "haqq/ucdao/MsgTransferWithRatio")
	legacy.RegisterAminoMsg(cdc, &MsgTransferOwnershipWithAmount{}, "haqq/ucdao/MsgTransferWithAmount")

	cdc.RegisterConcrete(Params{}, "haqq/x/ucdao/Params", nil)
}

// RegisterInterfaces registers the interfaces types with the Interface Registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgFund{},
		&MsgTransferOwnership{},
		&MsgTransferOwnershipWithRatio{},
		&MsgTransferOwnershipWithAmount{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)
}
