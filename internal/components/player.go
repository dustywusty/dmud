package components

import (
	"dmud/internal/common"
	"dmud/internal/util"
	"fmt"
	"strings"
	"sync"
)

type Player struct {
	sync.RWMutex

	Client common.Client

	Name string
	Area *Area

	// Command history and auto-complete
	CommandHistory *CommandHistory
	AutoComplete   *util.AutoComplete
}

func (p *Player) Broadcast(msg string) {
	p.Client.SendMessage(msg)
}

// Update the Player Look method
func (p *Player) Look(w WorldLike) {
	if p.Area == nil {
		p.Broadcast("You are nowhere.")
		return
	}

	// Area description
	p.Broadcast(p.Area.Description + "\n")
	if p.Area.Region != "" {
		p.Broadcast("Region: " + p.Area.Region)
	}

	// Show other players
	p.Area.PlayersMutex.RLock()
	var otherPlayers []string
	for _, player := range p.Area.Players {
		if player != p {
			otherPlayers = append(otherPlayers, player.Name)
		}
	}
	p.Area.PlayersMutex.RUnlock()
	for _, name := range otherPlayers {
		p.Broadcast(name + " is here.")
	}

	// Show NPCs after players
	npcs := p.Area.GetNPCs(w)
	for _, npc := range npcs {
		p.Broadcast(npc.Name + " is here.")
	}

	p.Broadcast("\n")

	// Show exits
	if len(p.Area.Exits) > 0 {
		exits := make([]string, len(p.Area.Exits))
		for i, exit := range p.Area.Exits {
			exits[i] = exit.Direction
		}
		p.Broadcast(fmt.Sprintf("\n\nExits: [%s]", strings.Join(exits, ", ")))
	}
}
