package common

type Client interface {
	CloseConnection() error
	GetMessage(maxLength int) (string, error)
	HandleRequest()
	SendMessage(msg string)
	RemoteAddr() string
}

type ClientType int

const (
	TCP ClientType = iota
	WebSocket
)
