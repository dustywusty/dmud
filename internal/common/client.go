package common

// -----------------------------------------------------------------------------

type ClientType int

const (
	TCP ClientType = iota
	WebSocket
)

// -----------------------------------------------------------------------------

type ConnectionStatus string

const (
	Connecting    ConnectionStatus = "connecting"
	Connected     ConnectionStatus = "connected"
	Disconnecting ConnectionStatus = "disconnecting"
	Disconnected  ConnectionStatus = "disconnected"
)

// -----------------------------------------------------------------------------

type Client interface {
	CloseConnection() error
	HandleRequest()
	SendMessage(msg string)
	RemoteAddr() string
}
