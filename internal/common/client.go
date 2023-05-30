package common

type Client interface {
	RemoteAddr() string
	CloseConnection() error
	SendMessage(msg string)
	GetMessage(maxLength int) (string, error)
	HandleRequest()
}
