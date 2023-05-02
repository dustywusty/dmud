package game

type Room struct {
	Description string
	ID          int
	Exits       map[string]*Room
	Name        string
	Players     map[string]Player
}

func NewRoom(id int, name string, description string) *Room {
	return &Room{
		Description: description,
		Exits:       make(map[string]*Room),
		ID:          id,
		Name:        name,
		Players:     make(map[string]Player),
	}
}

func (r *Room) AddExit(direction string, destination *Room) {
	r.Exits[direction] = destination
}

func (r *Room) AddPlayer(player Player) {
	r.Players[player.RemoteAddr()] = player
}

func (r *Room) RemovePlayer(player Player) {
	delete(r.Players, player.RemoteAddr())
}
