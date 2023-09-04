package components

import (
	"sync"

	"github.com/rs/zerolog/log"
)

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

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func (r *RoomComponent) AddPlayer(p *PlayerComponent) {
	r.Broadcast(p.Name + " enters")
	r.PlayersMutex.Lock()
	r.Players = append(r.Players, p)
	r.PlayersMutex.Unlock()
	log.Info().Msgf("Player added to room: %s", p.Name)
}

func (r *RoomComponent) GetExit(direction string) *Exit {
	for _, exit := range r.Exits {
		if exit.Direction == direction {
			return &exit
		}
	}
	return nil
}

func (r *RoomComponent) GetPlayer(name string) *PlayerComponent {
	r.PlayersMutex.Lock()
	for _, player := range r.Players {
		if player.Name == name {
			r.PlayersMutex.Unlock()
			return player
		}
	}
	r.PlayersMutex.Unlock()
	return nil
}

func (r *RoomComponent) Broadcast(msg string, exclude ...*PlayerComponent) {
	r.PlayersMutex.Lock()
	for _, player := range r.Players {
		if !contains(exclude, player) {
			player.Client.SendMessage(msg)
		}
	}
	r.PlayersMutex.Unlock()
	log.Info().Msgf("Broadcast: %s", msg)
}

func (r *RoomComponent) RemovePlayer(p *PlayerComponent) {
	r.PlayersMutex.Lock()
	for i, player := range r.Players {
		if player == p {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
	r.PlayersMutex.Unlock()
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
