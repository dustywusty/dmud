package common

type Client interface {
	CloseConnection() error
	HandleRequest()
	SendMessage(msg string)
	RemoteAddr() string
}

type ClientType int

const (
	TCP ClientType = iota
	WebSocket
)
