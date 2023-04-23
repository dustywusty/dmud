package game

type Player interface {
  RemoteAddr() string
  SendMessage(msg string)
  Name() string
	CloseConnection()
}
