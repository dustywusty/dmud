package game

type Room struct {
	ID      int
	Name    string
	Description string
	Exits   map[string]*Room
	Players map[string]Player
}

func NewRoom(id int, name string, description string) *Room {
	return &Room{
		ID:          id,
		Name:        name,
		Description: description,
		Exits:       make(map[string]*Room),
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
