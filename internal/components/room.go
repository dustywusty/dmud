package components

type Exit struct {
	Direction string
	RoomID    string
	Room      *RoomComponent
}

type RoomComponent struct {
	Description string
	Exits       []Exit
	Players     []*PlayerComponent
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

func (r *RoomComponent) AddPlayer(p *PlayerComponent) {
	r.MessageAllPlayers(p.Name + " enters")
	r.Players = append(r.Players, p)
}

func (r *RoomComponent) GetExit(direction string) *Exit {
	for _, exit := range r.Exits {
		if exit.Direction == direction {
			return &exit
		}
	}
	return nil
}

func (r *RoomComponent) GetPlayers() []*PlayerComponent {
	return r.Players
}

func (r *RoomComponent) MessageAllPlayers(msg string, exclude ...*PlayerComponent) {
	for _, player := range r.Players {
		if !contains(exclude, player) {
			player.Client.SendMessage(msg)
		}
	}
}

func (r *RoomComponent) RemovePlayer(p *PlayerComponent) {
	for i, player := range r.Players {
		if player == p {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
	r.MessageAllPlayers(p.Name + " leaves")
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Private
//

func contains(players []*PlayerComponent, player *PlayerComponent) bool {
	for _, p := range players {
		if p == player {
			return true
		}
	}
	return false
}
