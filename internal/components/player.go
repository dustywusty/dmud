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
	Room *Room

	// Command history and auto-complete
	CommandHistory *CommandHistory
	AutoComplete   *util.AutoComplete
}

func (p *Player) Broadcast(m string) {
	msg := m

	if !p.Client.SupportsPrompt() {
		trimmed := strings.TrimLeft(msg, "\n")

		switch {
		case trimmed == "" && strings.Contains(msg, "\n"):
			// Preserve intentional blank lines but collapse multiples to one.
			msg = "\n"
		case trimmed != "":
			msg = trimmed
		}
	}

	p.Client.SendMessage(msg)
}

// Update the Player Look method
func (p *Player) Look(w WorldLike) {
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
		p.Broadcast(fmt.Sprintf("\nExits: [%s]\n", strings.Join(exits, ", ")))
	}

	// Show other players
	p.Room.PlayersMutex.RLock()
	var otherPlayers []string
	for _, player := range p.Room.Players {
		if player != p {
			otherPlayers = append(otherPlayers, player.Name)
		}
	}
	p.Room.PlayersMutex.RUnlock()
	for _, name := range otherPlayers {
		p.Broadcast("  " + name + " is here.")
	}

	// Show NPCs after players
	npcs := p.Room.GetNPCs(w)
	for _, npc := range npcs {
		p.Broadcast("  " + npc.Name + " is here.")
	}
}
