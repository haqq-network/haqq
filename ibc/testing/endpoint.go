package ibctesting

import (
	"fmt"
	"strings"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibcgotesting "github.com/cosmos/ibc-go/v10/testing"

	"github.com/haqq-network/haqq/app"
)

// Endpoint is a which represents a channel endpoint and its associated
// client and connections. It contains client, connection, and channel
// configuration parameters. Endpoint functions will utilize the parameters
// set in the configuration structs when executing IBC messages.
type Endpoint struct {
	Chain        *ibcgotesting.TestChain
	Counterparty *Endpoint
	ClientID     string
	ConnectionID string
	ChannelID    string

	ClientConfig     ibcgotesting.ClientConfig
	ConnectionConfig *ibcgotesting.ConnectionConfig
	ChannelConfig    *ibcgotesting.ChannelConfig
}

// NewEndpoint constructs a new endpoint without the counterparty.
// CONTRACT: the counterparty endpoint must be set by the caller.
func NewEndpoint(
	chain *ibcgotesting.TestChain, clientConfig ibcgotesting.ClientConfig,
	connectionConfig *ibcgotesting.ConnectionConfig, channelConfig *ibcgotesting.ChannelConfig,
) *Endpoint {
	return &Endpoint{
		Chain:            chain,
		ClientConfig:     clientConfig,
		ConnectionConfig: connectionConfig,
		ChannelConfig:    channelConfig,
	}
}

// NewDefaultEndpoint constructs a new endpoint using default values.
// CONTRACT: the counterparty endpoitn must be set by the caller.
func NewDefaultEndpoint(chain *ibcgotesting.TestChain) *Endpoint {
	return &Endpoint{
		Chain:            chain,
		ClientConfig:     ibcgotesting.NewTendermintConfig(),
		ConnectionConfig: ibcgotesting.NewConnectionConfig(),
		ChannelConfig:    ibcgotesting.NewChannelConfig(),
	}
}

// QueryProof queries proof associated with this endpoint using the latest client state
// height on the counterparty chain.
func (endpoint *Endpoint) QueryProof(key []byte) ([]byte, clienttypes.Height) {
	// obtain the counterparty client representing the chain associated with the endpoint
	clientState := endpoint.Counterparty.Chain.GetClientState(endpoint.Counterparty.ClientID)

	// For Tendermint client state, access LatestHeight field directly
	tmClientState, ok := clientState.(*ibctm.ClientState)
	require.True(endpoint.Chain.TB, ok, "client state must be tendermint client state")

	// query proof on the counterparty using the latest height of the IBC client
	return endpoint.QueryProofAtHeight(key, tmClientState.LatestHeight.RevisionHeight)
}

// QueryProofAtHeight queries proof associated with this endpoint using the proof height
// provided
func (endpoint *Endpoint) QueryProofAtHeight(key []byte, height uint64) ([]byte, clienttypes.Height) {
	// query proof on the counterparty using the latest height of the IBC client
	return endpoint.Chain.QueryProofAtHeight(key, int64(height)) //nolint:gosec // won't overflow in normal conditions
}

// CreateClient creates an IBC client on the endpoint. It will update the
// clientID for the endpoint if the message is successfully executed.
// NOTE: a solo machine client will be created with an empty diversifier.
func (endpoint *Endpoint) CreateClient() (err error) {
	// ensure counterparty has committed state
	endpoint.Counterparty.Chain.NextBlock()

	var (
		clientState    exported.ClientState
		consensusState exported.ConsensusState
	)

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		tmConfig, ok := endpoint.ClientConfig.(*ibcgotesting.TendermintConfig)
		require.True(endpoint.Chain.TB, ok)

		header := endpoint.Counterparty.Chain.LatestCommittedHeader
		require.NotNil(endpoint.Chain.TB, header, "latest committed header must be set")
		height := header.GetHeight().(clienttypes.Height)
		clientState = ibctm.NewClientState(
			endpoint.Counterparty.Chain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
			height, commitmenttypes.GetSDKSpecs(), ibcgotesting.UpgradePath,
		)
		consensusState = header.ConsensusState()
	case exported.Solomachine:
		// TODO implement
		//		solo := NewSolomachine(endpoint.Chain.TB, endpoint.Chain.Codec, clientID, "", 1)
		//		clientState = solo.ClientState()
		//		consensusState = solo.ConsensusState()

	default:
		err = fmt.Errorf("client type %s is not supported", endpoint.ClientConfig.GetClientType())
	}

	if err != nil {
		return err
	}

	require.NotNil(
		endpoint.Chain.TB, endpoint.Chain.SenderAccount,
		fmt.Sprintf("expected sender account on chain with ID %q not to be nil", endpoint.Chain.ChainID),
	)

	zeroTimestamp := uint64(time.Time{}.UnixNano()) //nolint: gosec // converting zero value
	require.NotEqual(
		endpoint.Chain.TB, consensusState.GetTimestamp(), zeroTimestamp,
		"current timestamp on the last header is the zero time; it might be necessary to commit blocks with the IBC coordinator",
	)

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.TB, err)
	require.NoError(endpoint.Chain.TB, msg.ValidateBasic(), "failed to validate create client msg")

	res, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	if err != nil {
		return err
	}

	endpoint.ClientID, err = ibcgotesting.ParseClientIDFromEvents(res.GetEvents())
	require.NoError(endpoint.Chain.TB, err)

	return nil
}

// UpdateClient updates the IBC client associated with the endpoint.
func (endpoint *Endpoint) UpdateClient() (err error) {
	// ensure counterparty has committed state
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Counterparty.Chain)

	var header exported.ClientMessage

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		trustedHeight, ok := endpoint.Chain.GetClientLatestHeight(endpoint.ClientID).(clienttypes.Height)
		require.True(endpoint.Chain.TB, ok)
		header, err = endpoint.Counterparty.Chain.IBCClientHeader(endpoint.Counterparty.Chain.LatestCommittedHeader, trustedHeight)

	default:
		err = fmt.Errorf("client type %s is not supported", endpoint.ClientConfig.GetClientType())
	}

	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		endpoint.ClientID, header,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.TB, err)

	_, err = SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	return err
}

// UpgradeChain will upgrade a chain's chainID to the next revision number.
// It will also update the counterparty client.
// TODO: implement actual upgrade chain functionality via scheduling an upgrade
// and upgrading the client via MsgUpgradeClient
// see reference https://github.com/cosmos/ibc-go/pull/1169
func (endpoint *Endpoint) UpgradeChain() error {
	if strings.TrimSpace(endpoint.Counterparty.ClientID) == "" {
		return fmt.Errorf("cannot upgrade chain if there is no counterparty client")
	}

	clientState := endpoint.Counterparty.GetClientState().(*ibctm.ClientState)

	// increment revision number in chainID

	oldChainID := clientState.ChainId
	if !clienttypes.IsRevisionFormat(oldChainID) {
		return fmt.Errorf("cannot upgrade chain which is not of revision format: %s", oldChainID)
	}

	revisionNumber := clienttypes.ParseChainID(oldChainID)
	newChainID, err := clienttypes.SetRevisionNumber(oldChainID, revisionNumber+1)
	if err != nil {
		return err
	}

	// update chain
	haqqApp, isHaqq := endpoint.Chain.App.(*app.Haqq)
	if isHaqq {
		baseapp.SetChainID(newChainID)(haqqApp.GetBaseApp())
	} else {
		baseapp.SetChainID(newChainID)(endpoint.Chain.GetSimApp().GetBaseApp())
	}
	endpoint.Chain.ChainID = newChainID
	endpoint.Chain.ProposedHeader.ChainID = newChainID
	endpoint.Chain.NextBlock() // commit changes

	// update counterparty client manually
	clientState.ChainId = newChainID
	clientState.LatestHeight = clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.RevisionHeight+1)
	endpoint.Counterparty.SetClientState(clientState)

	header := endpoint.Chain.LatestCommittedHeader
	require.NotNil(endpoint.Chain.TB, header, "latest committed header must be set")
	consensusState := &ibctm.ConsensusState{
		Timestamp:          header.Header.Time,
		Root:               commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()),
		NextValidatorsHash: header.Header.NextValidatorsHash,
	}
	endpoint.Counterparty.SetConsensusState(consensusState, clientState.LatestHeight)

	// ensure the next update isn't identical to the one set in state
	endpoint.Chain.Coordinator.IncrementTime()
	endpoint.Chain.NextBlock()

	return endpoint.Counterparty.UpdateClient()
}

// ConnOpenInit will construct and execute a MsgConnectionOpenInit on the associated endpoint.
func (endpoint *Endpoint) ConnOpenInit() error {
	msg := connectiontypes.NewMsgConnectionOpenInit(
		endpoint.ClientID,
		endpoint.Counterparty.ClientID,
		endpoint.Counterparty.Chain.GetPrefix(), ibcgotesting.DefaultOpenInitVersion, endpoint.ConnectionConfig.DelayPeriod,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	if err != nil {
		return err
	}

	endpoint.ConnectionID, err = ibcgotesting.ParseConnectionIDFromEvents(res.GetEvents())
	require.NoError(endpoint.Chain.TB, err)

	return nil
}

// ConnOpenTry will construct and execute a MsgConnectionOpenTry on the associated endpoint.
func (endpoint *Endpoint) ConnOpenTry() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	initProof, proofHeight := endpoint.QueryConnectionHandshakeProof()

	msg := connectiontypes.NewMsgConnectionOpenTry(
		endpoint.ClientID, endpoint.Counterparty.ConnectionID, endpoint.Counterparty.ClientID,
		endpoint.Counterparty.Chain.GetPrefix(), []*connectiontypes.Version{ibcgotesting.ConnectionVersion},
		endpoint.ConnectionConfig.DelayPeriod,
		initProof, proofHeight,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	if err != nil {
		return err
	}

	if endpoint.ConnectionID == "" {
		endpoint.ConnectionID, err = ibcgotesting.ParseConnectionIDFromEvents(res.GetEvents())
		require.NoError(endpoint.Chain.TB, err)
	}

	return nil
}

// ConnOpenAck will construct and execute a MsgConnectionOpenAck on the associated endpoint.
func (endpoint *Endpoint) ConnOpenAck() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	tryProof, proofHeight := endpoint.QueryConnectionHandshakeProof()

	msg := connectiontypes.NewMsgConnectionOpenAck(
		endpoint.ConnectionID, endpoint.Counterparty.ConnectionID,
		tryProof, proofHeight,
		ibcgotesting.ConnectionVersion,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	_, err = SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	return err
}

// ConnOpenConfirm will construct and execute a MsgConnectionOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ConnOpenConfirm() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(connectionKey)

	msg := connectiontypes.NewMsgConnectionOpenConfirm(
		endpoint.ConnectionID,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	_, err = SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	return err
}

// QueryConnectionHandshakeProof returns all the proofs necessary to execute OpenTry or Open Ack of
// the connection handshakes. In ibc-go v10, this simplified to just return connection proof and height.
func (endpoint *Endpoint) QueryConnectionHandshakeProof() (
	connectionProof []byte, proofHeight clienttypes.Height,
) {
	// query proof for the connection on the counterparty
	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	connectionProof, proofHeight = endpoint.Counterparty.QueryProof(connectionKey)

	return connectionProof, proofHeight
}

// ChanOpenInit will construct and execute a MsgChannelOpenInit on the associated endpoint.
func (endpoint *Endpoint) ChanOpenInit() error {
	msg := channeltypes.NewMsgChannelOpenInit(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelConfig.Version, endpoint.ChannelConfig.Order, []string{endpoint.ConnectionID},
		endpoint.Counterparty.ChannelConfig.PortID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	if err != nil {
		return err
	}

	endpoint.ChannelID, err = ibcgotesting.ParseChannelIDFromEvents(res.GetEvents())
	require.NoError(endpoint.Chain.TB, err)

	// update version to selected app version
	// NOTE: this update must be performed after SendMsgs()
	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version
	endpoint.Counterparty.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry on the associated endpoint.
func (endpoint *Endpoint) ChanOpenTry() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenTry(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelConfig.Version, endpoint.ChannelConfig.Order, []string{endpoint.ConnectionID},
		endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID, endpoint.Counterparty.ChannelConfig.Version,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	if err != nil {
		return err
	}

	if endpoint.ChannelID == "" {
		endpoint.ChannelID, err = ibcgotesting.ParseChannelIDFromEvents(res.GetEvents())
		require.NoError(endpoint.Chain.TB, err)
	}

	// update version to selected app version
	// NOTE: this update must be performed after the endpoint channelID is set
	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version
	endpoint.Counterparty.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck on the associated endpoint.
func (endpoint *Endpoint) ChanOpenAck() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenAck(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		endpoint.Counterparty.ChannelID, endpoint.Counterparty.ChannelConfig.Version, // testing doesn't use flexible selection
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	if _, err = SendMsgs(endpoint.Chain, DefaultFeeAmt, msg); err != nil {
		return err
	}

	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ChanOpenConfirm() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenConfirm(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	_, err = SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	return err
}

// ChanCloseInit will construct and execute a MsgChannelCloseInit on the associated endpoint.
//
// NOTE: does not work with ibc-transfer module
func (endpoint *Endpoint) ChanCloseInit() error {
	msg := channeltypes.NewMsgChannelCloseInit(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	_, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, msg)
	return err
}

// SendPacket sends a packet through the channel keeper using the associated endpoint
// The counterparty client is updated so proofs can be sent to the counterparty chain.
// The packet sequence generated for the packet to be sent is returned. An error
// is returned if one occurs.
func (endpoint *Endpoint) SendPacket(
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	// In ibc-go v10, SendPacket no longer requires capability parameter
	// no need to send message, acting as a module
	sequence, err := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SendPacket(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return 0, err
	}

	// commit changes since no message was sent
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	err = endpoint.Counterparty.UpdateClient()
	if err != nil {
		return 0, err
	}

	return sequence, nil
}

// RecvPacket receives a packet on the associated endpoint.
// The counterparty client is updated.
func (endpoint *Endpoint) RecvPacket(packet channeltypes.Packet) error {
	_, err := endpoint.RecvPacketWithResult(packet)
	if err != nil {
		return err
	}

	return nil
}

// RecvPacketWithResult receives a packet on the associated endpoint and the result
// of the transaction is returned. The counterparty client is updated.
func (endpoint *Endpoint) RecvPacketWithResult(packet channeltypes.Packet) (*abci.ExecTxResult, error) {
	// get proof of packet commitment on source
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := endpoint.Counterparty.Chain.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	// receive on counterparty and update source client
	res, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, recvMsg)
	if err != nil {
		return nil, err
	}

	if err := endpoint.Counterparty.UpdateClient(); err != nil {
		return nil, err
	}

	return res, nil
}

// WriteAcknowledgement writes an acknowledgement on the channel associated with the endpoint.
// The counterparty client is updated.
func (endpoint *Endpoint) WriteAcknowledgement(ack exported.Acknowledgement, packet exported.PacketI) error {
	// In ibc-go v10, WriteAcknowledgement no longer requires capability parameter
	// no need to send message, acting as a handler
	err := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.WriteAcknowledgement(endpoint.Chain.GetContext(), packet, ack)
	if err != nil {
		return err
	}

	// commit changes since no message was sent
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	return endpoint.Counterparty.UpdateClient()
}

// AcknowledgePacket sends a MsgAcknowledgement to the channel associated with the endpoint.
func (endpoint *Endpoint) AcknowledgePacket(packet channeltypes.Packet, ack []byte) error {
	// get proof of acknowledgement on counterparty
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	_, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, ackMsg)
	return err
}

// TimeoutPacket sends a MsgTimeout to the channel associated with the endpoint.
func (endpoint *Endpoint) TimeoutPacket(packet channeltypes.Packet) error {
	// get proof for timeout based on channel order
	var packetKey []byte

	switch endpoint.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return fmt.Errorf("unsupported order type %s", endpoint.ChannelConfig.Order)
	}

	counterparty := endpoint.Counterparty
	proof, proofHeight := counterparty.QueryProof(packetKey)
	nextSeqRecv, found := counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(counterparty.Chain.GetContext(), counterparty.ChannelConfig.PortID, counterparty.ChannelID)
	require.True(endpoint.Chain.TB, found)

	timeoutMsg := channeltypes.NewMsgTimeout(
		packet, nextSeqRecv,
		proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	_, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, timeoutMsg)
	return err
}

// TimeoutOnClose sends a MsgTimeoutOnClose to the channel associated with the endpoint.
func (endpoint *Endpoint) TimeoutOnClose(packet channeltypes.Packet) error {
	// get proof for timeout based on channel order
	var packetKey []byte

	switch endpoint.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return fmt.Errorf("unsupported order type %s", endpoint.ChannelConfig.Order)
	}

	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	channelKey := host.ChannelKey(packet.GetDestPort(), packet.GetDestChannel())
	closedProof, _ := endpoint.Counterparty.QueryProof(channelKey)

	nextSeqRecv, found := endpoint.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(endpoint.Counterparty.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	require.True(endpoint.Chain.TB, found)

	// In ibc-go v10, channel upgrade sequence is no longer part of timeout on close
	timeoutOnCloseMsg := channeltypes.NewMsgTimeoutOnClose(
		packet, nextSeqRecv,
		proof, closedProof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	_, err := SendMsgs(endpoint.Chain, DefaultFeeAmt, timeoutOnCloseMsg)
	return err
}

// QueryChannelUpgradeProof, ChanUpgradeInit, ChanUpgradeTry are deprecated in ibc-go v10
// as channel upgrade functionality has been moved/removed.
// These methods are kept for backwards compatibility but will panic if called.
func (endpoint *Endpoint) QueryChannelUpgradeProof() {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
}

func (endpoint *Endpoint) ChanUpgradeInit() error {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
	return nil
}

func (endpoint *Endpoint) ChanUpgradeTry() error {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
	return nil
}

// ChanUpgradeAck, ChanUpgradeConfirm are deprecated in ibc-go v10
// as channel upgrade functionality has been moved/removed.
func (endpoint *Endpoint) ChanUpgradeAck() error {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
	return nil
}

func (endpoint *Endpoint) ChanUpgradeConfirm() error {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
	return nil
}

// ChanUpgradeOpen, ChanUpgradeTimeout, ChanUpgradeCancel are deprecated in ibc-go v10
// as channel upgrade functionality has been moved/removed.
// These methods are kept for backwards compatibility but will panic if called.
func (endpoint *Endpoint) ChanUpgradeOpen() error {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
	return nil
}

func (endpoint *Endpoint) ChanUpgradeTimeout() error {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
	return nil
}

func (endpoint *Endpoint) ChanUpgradeCancel() error {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
	return nil
}

// SetChannelState sets a channel state
func (endpoint *Endpoint) SetChannelState(state channeltypes.State) error {
	channel := endpoint.GetChannel()

	channel.State = state
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, channel)

	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	return endpoint.Counterparty.UpdateClient()
}

// GetClientState retrieves the Client State for this endpoint. The
// client state is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetClientState() exported.ClientState {
	return endpoint.Chain.GetClientState(endpoint.ClientID)
}

// SetClientState sets the client state for this endpoint.
func (endpoint *Endpoint) SetClientState(clientState exported.ClientState) {
	endpoint.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(endpoint.Chain.GetContext(), endpoint.ClientID, clientState)
}

// GetConsensusState retrieves the Consensus State for this endpoint at the provided height.
// The consensus state is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetConsensusState(height exported.Height) exported.ConsensusState {
	consensusState, found := endpoint.Chain.GetConsensusState(endpoint.ClientID, height)
	require.True(endpoint.Chain.TB, found)

	return consensusState
}

// SetConsensusState sets the consensus state for this endpoint.
func (endpoint *Endpoint) SetConsensusState(consensusState exported.ConsensusState, height exported.Height) {
	endpoint.Chain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(endpoint.Chain.GetContext(), endpoint.ClientID, height, consensusState)
}

// GetConnection retrieves an IBC Connection for the endpoint. The
// connection is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetConnection() connectiontypes.ConnectionEnd {
	connection, found := endpoint.Chain.App.GetIBCKeeper().ConnectionKeeper.GetConnection(endpoint.Chain.GetContext(), endpoint.ConnectionID)
	require.True(endpoint.Chain.TB, found)

	return connection
}

// SetConnection sets the connection for this endpoint.
func (endpoint *Endpoint) SetConnection(connection connectiontypes.ConnectionEnd) {
	endpoint.Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(endpoint.Chain.GetContext(), endpoint.ConnectionID, connection)
}

// GetChannel retrieves an IBC Channel for the endpoint. The channel
// is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetChannel() channeltypes.Channel {
	channel, found := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	require.True(endpoint.Chain.TB, found)

	return channel
}

// SetChannel sets the channel for this endpoint.
func (endpoint *Endpoint) SetChannel(channel channeltypes.Channel) {
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, channel)
}

// GetChannelUpgrade, SetChannelUpgrade, and SetChannelCounterpartyUpgrade are deprecated
// in ibc-go v10 as channel upgrade functionality has been moved/removed.
// These methods are kept for backwards compatibility but will panic if called.
func (endpoint *Endpoint) GetChannelUpgrade() {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
}

func (endpoint *Endpoint) SetChannelUpgrade(_ interface{}) {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
}

func (endpoint *Endpoint) SetChannelCounterpartyUpgrade(_ interface{}) {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
}

// QueryClientStateProof performs and abci query for a client stat associated
// with this endpoint and returns the ClientState along with the proof.
func (endpoint *Endpoint) QueryClientStateProof() (exported.ClientState, []byte) {
	// retrieve client state to provide proof for
	clientState := endpoint.GetClientState()

	clientKey := host.FullClientStateKey(endpoint.ClientID)
	clientProof, _ := endpoint.QueryProof(clientKey)

	return clientState, clientProof
}

// GetProposedUpgrade is deprecated in ibc-go v10 as channel upgrade functionality has been moved/removed.
// This method is kept for backwards compatibility but will panic if called.
func (endpoint *Endpoint) GetProposedUpgrade() {
	endpoint.Chain.TB.Fatal("channel upgrade functionality is not available in ibc-go v10")
}
