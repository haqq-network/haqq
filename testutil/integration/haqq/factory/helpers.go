package factory

import (
	errorsmod "cosmossdk.io/errors"
	amino "github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	testutiltypes "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	enccodec "github.com/haqq-network/haqq/encoding/codec"
	"github.com/haqq-network/haqq/testutil/tx"
	haqqtypes "github.com/haqq-network/haqq/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// buildMsgEthereumTx builds an Ethereum transaction from the given arguments and populates the From field.
func buildMsgEthereumTx(txArgs evmtypes.EvmTxArgs, fromAddr common.Address) evmtypes.MsgEthereumTx {
	msgEthereumTx := evmtypes.NewTx(&txArgs)
	msgEthereumTx.From = fromAddr.String()
	return *msgEthereumTx
}

// signMsgEthereumTx signs a MsgEthereumTx with the provided private key and chainID.
func signMsgEthereumTx(msgEthereumTx evmtypes.MsgEthereumTx, privKey cryptotypes.PrivKey, chainID string) (evmtypes.MsgEthereumTx, error) {
	ethChainID, err := haqqtypes.ParseChainID(chainID)
	if err != nil {
		return evmtypes.MsgEthereumTx{}, errorsmod.Wrapf(err, "failed to parse chainID: %v", chainID)
	}

	signer := ethtypes.LatestSignerForChainID(ethChainID)
	err = msgEthereumTx.Sign(signer, tx.NewSigner(privKey))
	if err != nil {
		return evmtypes.MsgEthereumTx{}, errorsmod.Wrap(err, "failed to sign transaction")
	}

	// Validate the transaction to avoid unrealistic behavior
	if err = msgEthereumTx.ValidateBasic(); err != nil {
		return evmtypes.MsgEthereumTx{}, errorsmod.Wrap(err, "failed to validate transaction")
	}
	return msgEthereumTx, nil
}

// makeConfig creates an EncodingConfig for testing
func makeConfig(mb module.BasicManager) testutiltypes.TestEncodingConfig {
	cdc := amino.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	codec := amino.NewProtoCodec(interfaceRegistry)

	encodingConfig := testutiltypes.TestEncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
		TxConfig:          authtx.NewTxConfig(codec, authtx.DefaultSignModes),
		Amino:             cdc,
	}

	enccodec.RegisterLegacyAminoCodec(encodingConfig.Amino)
	mb.RegisterLegacyAminoCodec(encodingConfig.Amino)
	enccodec.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	mb.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
