package components

import "sync"

type Exit struct {
	Direction string
	RoomID    string
	Room      *RoomComponent
}

type RoomComponent struct {
	Description  string
	Exits        []Exit
	Players      []*PlayerComponent
	PlayersMutex sync.RWMutex
}

// /////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func (r *RoomComponent) AddPlayer(p *PlayerComponent) {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	r.Broadcast(p.Name + " enters")
	r.Players = append(r.Players, p)
}

func (r *RoomComponent) GetExit(direction string) *Exit {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	for _, exit := range r.Exits {
		if exit.Direction == direction {
			return &exit
		}
	}
	return nil
}

func (r *RoomComponent) Broadcast(msg string, exclude ...*PlayerComponent) {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	for _, player := range r.Players {
		if !contains(exclude, player) {
			player.Client.SendMessage(msg)
		}
	}
}

func (r *RoomComponent) RemovePlayer(p *PlayerComponent) {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	for i, player := range r.Players {
		if player == p {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
	r.Broadcast(p.Name + " leaves")
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func contains(players []*PlayerComponent, player *PlayerComponent) bool {
	for _, p := range players {
		if p == player {
			return true
		}
	}
	return false
}
