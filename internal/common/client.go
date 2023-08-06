package common

type ClientType int

const (
	TCP ClientType = iota
	WebSocket
)

type Client interface {
	CloseConnection() error
	HandleRequest()
	SendMessage(msg string)
	RemoteAddr() string
}

type EntityID string
