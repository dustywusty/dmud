package game

import (
	"fmt"
	"strings"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
)

// playerShift returns the player's active beast form, or nil if unshifted.
func (g *Game) playerShift(entityID common.EntityID) *components.Shift {
	if c, err := g.world.GetComponent(entityID, "Shift"); err == nil {
		if s, ok := c.(*components.Shift); ok {
			return s
		}
	}
	return nil
}

// playerMeleeDamage is the player's natural-weapon damage range — a beast form's
// claws/teeth when shifted, or the unarmed default otherwise. Strength scales it
// in performAttack, just like any melee.
func (g *Game) playerMeleeDamage(player *components.Player) (int, int) {
	if eid, err := g.getPlayerEntity(player); err == nil {
		if sh := g.playerShift(eid); sh != nil {
			if form, ok := components.FormFor(sh.Form); ok {
				return form.MinDamage, form.MaxDamage
			}
		}
	}
	return 10, 50
}

func (g *Game) handleShift(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Something went wrong.")
		return
	}

	if len(args) == 0 {
		if sh := g.playerShift(entityID); sh != nil {
			if form, ok := components.FormFor(sh.Form); ok {
				player.Broadcast(fmt.Sprintf("You are currently %s. (use 'revert' to return)", form.Name))
				return
			}
		}
		player.Broadcast("Shift into what? Forms: " + strings.Join(components.FormNames(), ", "))
		return
	}

	if !classAllowsAbility(g.getStats(entityID), "shapeshift") {
		player.Broadcast("Only Druids can take beast shape.")
		return
	}

	form, ok := components.FormFor(args[0])
	if !ok {
		player.Broadcast("You don't know that form. Forms: " + strings.Join(components.FormNames(), ", "))
		return
	}
	if lvl := g.casterLevel(entityID); lvl < form.MinLevel {
		player.Broadcast(fmt.Sprintf("You can't hold the shape of %s yet -- it requires level %d (you are level %d).",
			form.Name, form.MinLevel, lvl))
		return
	}

	g.playersMu.RLock()
	pe := g.players[player.Name]
	g.playersMu.RUnlock()
	if pe == nil {
		player.Broadcast("Something went wrong.")
		return
	}
	g.world.AddComponent(pe, &components.Shift{Form: form.Key})

	// If already mid-fight, the new claws apply immediately.
	if combat, err := ecs.GetTypedComponent[*components.Combat](g.world, entityID, "Combat"); err == nil && combat != nil {
		combat.MinDamage = form.MinDamage
		combat.MaxDamage = form.MaxDamage
	}

	player.Broadcast(fmt.Sprintf("Your body twists and swells -- you take the form of %s!", form.Name))
	if player.Area != nil {
		player.Area.Broadcast(fmt.Sprintf("%s shifts into %s.", player.Name, form.Name), player)
	}
	player.BroadcastState(g.world.AsWorldLike(), entityID)
}

func (g *Game) handleRevert(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Something went wrong.")
		return
	}
	if g.playerShift(entityID) == nil {
		player.Broadcast("You're already in your own form.")
		return
	}

	g.world.RemoveComponent(entityID, "Shift")
	if combat, err := ecs.GetTypedComponent[*components.Combat](g.world, entityID, "Combat"); err == nil && combat != nil {
		combat.MinDamage = 10
		combat.MaxDamage = 50
	}

	player.Broadcast("The beast shape sloughs away and you stand on two legs again.")
	if player.Area != nil {
		player.Area.Broadcast(fmt.Sprintf("%s returns to their own form.", player.Name), player)
	}
	player.BroadcastState(g.world.AsWorldLike(), entityID)
}
