package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// RegisterLegacyAminoCodec registers the necessary x/ethiq interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgMintHaqq{}, "haqq/ethiq/MsgMintHaqq", nil)
	cdc.RegisterConcrete(&MsgMintHaqqByApplication{}, "haqq/ethiq/MsgMintHaqqByApplication", nil)

	cdc.RegisterConcrete(&MintHaqqAuthorization{}, "haqq/ethiq/MintHaqqAuthorization", nil)
	cdc.RegisterConcrete(&MintHaqqByApplicationIDAuthorization{}, "haqq/ethiq/MintHaqqByApplicationIDAuthorization", nil)
}

// RegisterInterfaces registers the x/ethiq interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgMintHaqq{},
		&MsgMintHaqqByApplication{},
	)
	registry.RegisterImplementations(
		(*authz.Authorization)(nil),
		&MintHaqqAuthorization{},
		&MintHaqqByApplicationIDAuthorization{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
