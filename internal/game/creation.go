package game

import (
	"fmt"
	"strings"

	"dmud/internal/common"
	"dmud/internal/components"
)

// ghostCommands are the only commands an un-manifested ghost may use: the
// character-creation choices plus harmless info commands.
var ghostCommands = map[string]bool{
	"name": true, "race": true, "class": true, "classes": true,
	"look": true, "who": true, "help": true, "commands": true,
	"score": true, "stats": true, "examine": true,
	"quit": true, "save": true, "login": true,
}

func (g *Game) getCreation(entityID common.EntityID) *components.Creation {
	if c, err := g.world.GetComponent(entityID, "Creation"); err == nil {
		if cr, ok := c.(*components.Creation); ok {
			return cr
		}
	}
	return nil
}

// ensureCreation attaches a Creation tracker to an un-manifested player so the
// ghost flow can run. No-op for already-created characters.
func (g *Game) ensureCreation(entityID common.EntityID, player *components.Player) {
	if player == nil || player.Created || g.getCreation(entityID) != nil {
		return
	}
	if entity, err := g.world.FindEntity(entityID); err == nil {
		g.world.AddComponent(&entity, components.NewCreation())
	}
}

// ghostPrompt tells a ghost which creation choices remain.
func (g *Game) ghostPrompt(player *components.Player, entityID common.EntityID) {
	cr := g.getCreation(entityID)
	if cr == nil {
		return
	}
	var todo []string
	if !cr.NamePicked {
		todo = append(todo, "a name ('name <name>')")
	}
	if !cr.RacePicked {
		todo = append(todo, "a race ('race <race>')")
	}
	if !cr.ClassPicked {
		todo = append(todo, "a class ('class <class>')")
	}
	player.Broadcast("Still to choose: " + strings.Join(todo, ", ") + ".")
	player.Broadcast("(Type 'race' or 'classes' to see your options.)")
}

// ghostIntro greets a freshly-spawned ghost.
func (g *Game) ghostIntro(player *components.Player, entityID common.EntityID) {
	player.Broadcast("")
	player.Broadcast("You drift into the world as a formless spirit -- barely real, unseen.")
	player.Broadcast("To take flesh you must choose a name, a race, and a class.")
	g.ghostPrompt(player, entityID)
}

// creationStep records one creation choice for a ghost, manifesting them once
// all three are made. No-op for already-created characters.
func (g *Game) creationStep(player *components.Player, step string) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}
	cr := g.getCreation(entityID)
	if cr == nil {
		return
	}
	switch step {
	case "name":
		cr.NamePicked = true
	case "race":
		cr.RacePicked = true
	case "class":
		cr.ClassPicked = true
	}
	if cr.Done() {
		g.manifest(player, entityID)
	} else {
		g.ghostPrompt(player, entityID)
	}
}

// manifest turns a finished ghost into a real, playable character.
func (g *Game) manifest(player *components.Player, entityID common.EntityID) {
	player.Created = true
	g.world.RemoveComponent(entityID, "Creation")

	race, class := "wanderer", "adventurer"
	if stats := g.getStats(entityID); stats != nil {
		if stats.Race != "" {
			race = stats.Race
		}
		if def, ok := classByKey(stats.Class); ok {
			class = def.Name
		}
	}

	player.Broadcast("")
	player.Broadcast("Light floods through you and your form snaps into being!")
	player.Broadcast(fmt.Sprintf("You are %s, a %s %s. The world is yours.", player.Name, race, class))
	if player.Area != nil {
		player.Area.Broadcast(fmt.Sprintf("%s coalesces into being.", player.Name), player)
	}
	player.Look(g.world.AsWorldLike())
	player.BroadcastState(g.world.AsWorldLike(), entityID)
}
