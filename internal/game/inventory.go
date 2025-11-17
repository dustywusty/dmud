package game

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

func (g *Game) handleLoot(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Loot what? Usage: loot <corpse>")
		return
	}

	targetName := strings.ToLower(strings.Join(args, " "))

	// Find corpses in the area
	corpses, err := g.world.FindEntitiesByComponentPredicate("Corpse", func(i interface{}) bool {
		c, ok := i.(*components.Corpse)
		return ok && c.Area == player.Area
	})

	if err != nil || len(corpses) == 0 {
		player.Broadcast("There are no corpses here to loot.")
		return
	}

	// Find the matching corpse
	var targetCorpse *components.Corpse
	for _, corpseEntity := range corpses {
		corpseComp, err := g.world.GetComponent(corpseEntity.ID, "Corpse")
		if err != nil {
			continue
		}

		corpse := corpseComp.(*components.Corpse)
		corpseName := strings.ToLower(corpse.GetDescription())

		if strings.Contains(corpseName, targetName) {
			targetCorpse = corpse
			break
		}
	}

	if targetCorpse == nil {
		player.Broadcast("You don't see that corpse here.")
		return
	}

	// Get player's inventory
	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}

	playerInvComp, err := g.world.GetComponent(playerEntity, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory!")
		return
	}
	playerInventory := playerInvComp.(*components.Inventory)

	// Loot all items from corpse
	if targetCorpse.Inventory == nil {
		player.Broadcast("The corpse has nothing to loot.")
		return
	}

	targetCorpse.Inventory.Lock()
	defer targetCorpse.Inventory.Unlock()

	if len(targetCorpse.Inventory.Items) == 0 {
		player.Broadcast("The corpse has nothing to loot.")
		return
	}

	lootedItems := make([]string, 0)
	for _, item := range targetCorpse.Inventory.Items {
		if playerInventory.IsFull() {
			player.Broadcast("Your inventory is full!")
			break
		}

		if playerInventory.AddItem(item.Clone()) {
			lootedItems = append(lootedItems, item.Name)
		}
	}

	// Clear corpse inventory
	targetCorpse.Inventory.Items = make([]*components.Item, 0)

	if len(lootedItems) > 0 {
		player.Broadcast(fmt.Sprintf("You looted: %s", strings.Join(lootedItems, ", ")))
		player.Area.Broadcast(fmt.Sprintf("%s loots %s.", player.Name, targetCorpse.GetDescription()), player)
	} else {
		player.Broadcast("You couldn't loot anything.")
	}
}

func (g *Game) handleInventory(player *components.Player, args []string, game *Game) {
	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}

	invComp, err := g.world.GetComponent(playerEntity, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory!")
		return
	}

	inventory := invComp.(*components.Inventory)
	items := inventory.GetItems()

	if len(items) == 0 {
		player.Broadcast("Your inventory is empty.")
		return
	}

	var output strings.Builder
	output.WriteString("Inventory:\n")

	for _, item := range items {
		if item.Stackable && item.Quantity > 1 {
			output.WriteString(fmt.Sprintf("  %s x%d\n", item.Name, item.Quantity))
		} else {
			output.WriteString(fmt.Sprintf("  %s\n", item.Name))
		}
	}

	inventory.RLock()
	if inventory.MaxSlots > 0 {
		output.WriteString(fmt.Sprintf("(%d/%d slots used)", len(inventory.Items), inventory.MaxSlots))
	}
	inventory.RUnlock()

	player.Broadcast(output.String())
}

func (g *Game) handleGet(player *components.Player, args []string, game *Game) {
	player.Broadcast("Item pickup from ground not yet implemented. Use 'loot' for corpses.")
}

func (g *Game) handleDrop(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Drop what? Usage: drop <item>")
		return
	}

	itemName := strings.ToLower(strings.Join(args, " "))

	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}

	invComp, err := g.world.GetComponent(playerEntity, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory!")
		return
	}

	inventory := invComp.(*components.Inventory)
	items := inventory.GetItems()

	// Find matching item
	var targetItem *components.Item
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), itemName) {
			targetItem = item
			break
		}
	}

	if targetItem == nil {
		player.Broadcast("You don't have that item.")
		return
	}

	// Remove item from inventory (quantity 1 if stackable)
	quantity := 1
	if !targetItem.Stackable {
		quantity = targetItem.Quantity
	}

	removed := inventory.RemoveItem(targetItem.ID, quantity)
	if removed != nil {
		player.Broadcast(fmt.Sprintf("You dropped %s.", removed.Name))
		player.Area.Broadcast(fmt.Sprintf("%s dropped %s.", player.Name, removed.Name), player)
		// TODO: Add item to ground/area when we have ground items system
	} else {
		player.Broadcast("Failed to drop item.")
	}
}

func (g *Game) getPlayerEntity(player *components.Player) (common.EntityID, error) {
	g.playersMu.RLock()
	defer g.playersMu.RUnlock()

	for _, playerEntity := range g.players {
		playerComp, err := g.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			continue
		}

		p, ok := playerComp.(*components.Player)
		if !ok {
			continue
		}

		if p == player {
			return playerEntity.ID, nil
		}
	}

	return "", fmt.Errorf("player entity not found")
}
