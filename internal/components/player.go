package components

import (
	"dmud/internal/common"
	"strings"
	"sync"
)

type Player struct {
	sync.RWMutex

	Client common.Client

	Name string
	Room *Room
}

func (p *Player) Broadcast(m string) {
	p.Client.SendMessage(m + "\n\n")
}

// Update the Player Look method
func (p *Player) Look(w ecs.WorldLike) {
    if p.Room == nil {
        p.Broadcast("You are nowhere.")
        return
    }

    // Room description
    p.Broadcast(p.Room.Description)

    // Show exits
    if len(p.Room.Exits) > 0 {
        exits := make([]string, len(p.Room.Exits))
        for i, exit := range p.Room.Exits {
            exits[i] = exit.Direction
        }
        p.Broadcast("Exits: " + strings.Join(exits, ", "))
    }

    // Show other players
    p.Room.PlayersMutex.RLock()
    for _, player := range p.Room.Players {
        if player != p {
            p.Broadcast("  " + player.Name + " is here.")
        }
    }
    p.Room.PlayersMutex.RUnlock()

    // Show NPCs
    npcs := p.Room.GetNPCs(w)
    for _, npc := range npcs {
        p.Broadcast("  " + npc.Name + " is here.")
    }
}