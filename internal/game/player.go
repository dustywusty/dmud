package game

type Player struct {
	ID       int
	Position *PositionComponent
	RemoteAddr() string
	SendMessage(msg string)
	Name() string
	CloseConnection()
}

func NewPlayer(id int, roomID string) *Player {
	return &Player{
		ID:       id,
		Position: &PositionComponent{RoomID: roomID},
		// Initialize other components...
	}
}
