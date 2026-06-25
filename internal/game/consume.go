package game

import (
	"dmud/internal/components"
	"fmt"
	"strings"
)

// handleConsume powers eat/drink/quaff/use: it removes one matching consumable
// from the player's inventory and applies its health/endurance restore.
func (g *Game) handleConsume(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Consume what? Try: eat bread · drink draught · quaff potion")
		return
	}
	query := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))

	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Something went wrong.")
		return
	}
	invComp, err := g.world.GetComponent(entityID, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory!")
		return
	}
	inv := invComp.(*components.Inventory)

	item, eff := findConsumable(inv, query)
	if item == nil {
		player.Broadcast("You have nothing like that to consume.")
		return
	}
	if inv.RemoveItem(item.ID, 1) == nil {
		player.Broadcast("You reach for it, but it's already gone.")
		return
	}

	var restored []string
	if eff.RestoreHP > 0 {
		if hpComp, err := g.world.GetComponent(entityID, "Health"); err == nil {
			if h, ok := hpComp.(*components.Health); ok {
				before := h.Current
				h.Heal(eff.RestoreHP)
				if gained := h.Current - before; gained > 0 {
					restored = append(restored, fmt.Sprintf("%d health", gained))
				}
			}
		}
	}
	if eff.RestoreEP > 0 {
		if end := getEndurance(g, entityID); end != nil {
			if gained := end.Restore(eff.RestoreEP); gained > 0 {
				restored = append(restored, fmt.Sprintf("%d endurance", gained))
			}
		}
	}

	player.Broadcast(eff.SelfMessage)
	if len(restored) > 0 {
		player.Broadcast("You recover " + strings.Join(restored, " and ") + ".")
	} else {
		player.Broadcast("You feel no different — you were already at your peak.")
	}
	if player.Area != nil && eff.AreaMessage != "" {
		player.Area.Broadcast(fmt.Sprintf(eff.AreaMessage, player.Name), player)
	}

	player.BroadcastState(g.world.AsWorldLike(), entityID)
}

// findConsumable resolves a typed query to a consumable in the inventory,
// matching by item ID or a substring of the item's display name.
func findConsumable(inv *components.Inventory, query string) (*components.Item, *components.ConsumableEffect) {
	qID := strings.ReplaceAll(query, " ", "_")
	for _, it := range inv.GetItems() {
		eff, ok := components.ConsumableFor(it.ID)
		if !ok {
			continue
		}
		if it.ID == query || it.ID == qID || strings.Contains(strings.ToLower(it.Name), query) {
			return it, eff
		}
	}
	return nil, nil
}
