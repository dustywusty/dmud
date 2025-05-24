package components

import (
	"dmud/internal/util"
	"sync"

	"github.com/rs/zerolog/log"
)

type Exit struct {
	Direction string
	RoomID    string
	Room      *Room
}

type Room struct {
	sync.RWMutex

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

	r.Broadcast(p.Name + " enters")

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

func (r *Room) GetNPCs(w util.WorldLike) []*NPC {
    r.RLock()
    defer r.RUnlock()

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
        npcComponent, err := util.GetTypedComponent[*NPC](w, entity.ID, "NPC")
        if err == nil {
            npcs = append(npcs, npcComponent)
        }
    }

    return npcs
}

func (r *Room) GetPlayer(name string) *Player {
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
	for i, player := range r.Players {
		if player == p {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
	r.PlayersMutex.Unlock()
	r.Broadcast(p.Name + " leaves")
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
