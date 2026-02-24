package ethiqv1

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	protov2 "google.golang.org/protobuf/proto"
)

// GetSignersMintHaqq gets the signer's address from the Ethereum tx signature for MsgMintHaqq
func GetSignersMintHaqq(msg protov2.Message) ([][]byte, error) {
	msgMintHaqq, ok := msg.(*MsgMintHaqq)
	if !ok {
		return nil, fmt.Errorf("invalid type, expected MsgMintHaqq and got %T", msg)
	}

	// The sender on the msg is a hex address
	var sender []byte
	if common.IsHexAddress(msgMintHaqq.FromAddress) {
		sender = common.HexToAddress(msgMintHaqq.FromAddress).Bytes()
	} else {
		senderBech32, err := sdk.AccAddressFromBech32(msgMintHaqq.FromAddress)
		if err != nil {
			return nil, err
		}
		sender = senderBech32.Bytes()
	}

	return [][]byte{sender}, nil
}
