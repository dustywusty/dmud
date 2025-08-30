package components

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type Exit struct {
	Direction string
	RoomID    string
	Room      *Room
}

type Room struct {
	X int
	Y int
	Z int

	Description string
	Exits       []Exit
	Players     []*Player

	PlayersMutex sync.RWMutex
}

func (r *Room) AddPlayer(p *Player) {
	log.Info().Msgf("Player added to room: %s", p.Name)

	r.Broadcast("\n" + p.Name + " enters")

	r.PlayersMutex.Lock()
	r.Players = append(r.Players, p)
	r.PlayersMutex.Unlock()
}

func (r *Room) GetExit(direction string) *Exit {
	for _, exit := range r.Exits {
		if exit.Direction == direction {
			return &exit
		}
	}
	return nil
}

func (r *Room) GetNPCs(w WorldLike) []*NPC {
	var npcs []*NPC

	// Find all NPCs in this room
	entities, err := w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		npc, ok := i.(*NPC)
		return ok && npc.Room == r
	})

	if err != nil {
		return npcs
	}

	for _, entity := range entities {
		npcComponent, err := w.GetComponent(entity.GetID(), "NPC")
		if err == nil {
			if npc, ok := npcComponent.(*NPC); ok {
				npcs = append(npcs, npc)
			}
		}
	}

	return npcs
}

func (r *Room) GetPlayer(name string) *Player {
	r.PlayersMutex.RLock()
	defer r.PlayersMutex.RUnlock()
	
	for _, player := range r.Players {
		if player.Name == name {
			return player
		}
	}
	return nil
}

func (r *Room) Broadcast(msg string, exclude ...*Player) {
	r.PlayersMutex.Lock()
	defer r.PlayersMutex.Unlock()

	if len(r.Players) == 0 {
		return
	}

	for _, player := range r.Players {
		if !contains(exclude, player) {
			player.Broadcast(msg)
		}
	}
}

func (r *Room) RemovePlayer(p *Player) {
	r.PlayersMutex.Lock()
	removed := false
	for i, player := range r.Players {
		if player == p {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			removed = true
			break
		}
	}
	r.PlayersMutex.Unlock()
	
	if removed {
		r.Broadcast("\n" + p.Name + " leaves")
	}
}

// ..

func contains(players []*Player, player *Player) bool {
	for _, p := range players {
		if p == player {
			return true
		}
	}
	return false
}
