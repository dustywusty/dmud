package components

import (
	"dmud/internal/common"
	"dmud/internal/util"
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

func (p *Player) Look(w WorldLike) {
	p.Broadcast(p.DescribeArea(w))
}

// DescribeArea returns information about the player's current area, including
// other players, NPCs, and exits.
func (p *Player) DescribeArea(w WorldLike) string {
	if p.Area == nil {
		return "You are nowhere."
	}

	var b strings.Builder

	b.WriteString(p.Area.Description)

	p.Area.PlayersMutex.RLock()
	var otherPlayers []string
	for _, player := range p.Area.Players {
		if player != p {
			otherPlayers = append(otherPlayers, player.Name)
		}
	}
	p.Area.PlayersMutex.RUnlock()

	npcs := p.Area.GetNPCs(w)
	corpses := p.Area.GetCorpses(w)

	hasEntities := len(otherPlayers) > 0 || len(npcs) > 0 || len(corpses) > 0
	if hasEntities {
		b.WriteString("\n\n")
		for _, name := range otherPlayers {
			b.WriteString(name)
			b.WriteString(" is here.\n")
		}

		for _, npc := range npcs {
			b.WriteString(npc.Name)
			b.WriteString(" is here.\n")
		}

		for _, corpse := range corpses {
			b.WriteString(corpse.GetDescription())
			b.WriteString(" is here.\n")
		}
	}

	if len(p.Area.Exits) > 0 {
		exits := make([]string, len(p.Area.Exits))
		for i, exit := range p.Area.Exits {
			exits[i] = exit.Direction
		}
		if hasEntities {
			b.WriteString("\n")
		} else {
			b.WriteString("\n\n")
		}
		b.WriteString("Exits: [")
		b.WriteString(strings.Join(exits, ", "))
		b.WriteString("]\n")
	}

	return b.String()
}
