package components

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type Exit struct {
	Direction string
	AreaID    string
	Area      *Area
}

type Area struct {
	X int
	Y int
	Z int

	Region      string
	Description string
	Exits       []Exit
	Players     []*Player

	PlayersMutex sync.RWMutex
}

func (a *Area) AddPlayer(p *Player) {
	log.Info().Msgf("Player added to area: %s", p.Name)

	a.Broadcast(p.Name + " enters")

	a.PlayersMutex.Lock()
	a.Players = append(a.Players, p)
	a.PlayersMutex.Unlock()
}

func (a *Area) GetExit(direction string) *Exit {
	for i := range a.Exits {
		exit := &a.Exits[i]
		if exit.Direction == direction {
			return exit
		}
	}
	return nil
}

func (a *Area) GetNPCs(w WorldLike) []*NPC {
	var npcs []*NPC

	entities, err := w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		npc, ok := i.(*NPC)
		return ok && npc.Area == a
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

func (a *Area) GetCorpses(w WorldLike) []*Corpse {
	var corpses []*Corpse

	entities, err := w.FindEntitiesByComponentPredicate("Corpse", func(i interface{}) bool {
		corpse, ok := i.(*Corpse)
		return ok && corpse.Area == a
	})

	if err != nil {
		return corpses
	}

	for _, entity := range entities {
		corpseComponent, err := w.GetComponent(entity.GetID(), "Corpse")
		if err == nil {
			if corpse, ok := corpseComponent.(*Corpse); ok {
				corpses = append(corpses, corpse)
			}
		}
	}

	return corpses
}

func (a *Area) GetPlayer(name string) *Player {
	a.PlayersMutex.RLock()
	defer a.PlayersMutex.RUnlock()

	for _, player := range a.Players {
		if player.Name == name {
			return player
		}
	}
	return nil
}

func (a *Area) Broadcast(msg string, exclude ...*Player) {
	a.PlayersMutex.Lock()
	defer a.PlayersMutex.Unlock()

	if len(a.Players) == 0 {
		return
	}

	for _, player := range a.Players {
		if !contains(exclude, player) {
			player.Broadcast(msg)
		}
	}
}

func (a *Area) RemovePlayer(p *Player) {
	a.PlayersMutex.Lock()
	removed := false
	for i, player := range a.Players {
		if player == p {
			a.Players = append(a.Players[:i], a.Players[i+1:]...)
			removed = true
			break
		}
	}
	a.PlayersMutex.Unlock()

	if removed {
		a.Broadcast(p.Name + " leaves")
	}
}

func contains(players []*Player, player *Player) bool {
	for _, p := range players {
		if p == player {
			return true
		}
	}
	return false
}
