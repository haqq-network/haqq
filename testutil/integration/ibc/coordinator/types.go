package coordinator

// Endpoint defines the identifiers for a chain's client, connection, and channel.
type Endpoint struct {
	ChainID      string
	ClientID     string
	ConnectionID string
	ChannelID    string
	PortID       string
}

// IBCConnection defines the connection between two chains.
type IBCConnection struct {
	EndpointA Endpoint
	EndpointB Endpoint
}

func NewEndpoint() Endpoint {
	return Endpoint{}
}

func NewIBCConnection(a, b Endpoint) IBCConnection {
	return IBCConnection{
		EndpointA: a,
		EndpointB: b,
	}
}
