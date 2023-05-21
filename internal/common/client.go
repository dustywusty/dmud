package common

type Client interface {
	ID() string
	CloseConnection() error
	SendMessage(msg string)
}
