package game

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func (g *Game) handleLoot(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Loot what? Usage: loot <corpse> or loot all")
		return
	}

	targetName := strings.ToLower(strings.Join(args, " "))

	// Check for "loot all"
	if targetName == "all" {
		g.handleLootAll(player, game)
		return
	}

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

	items := targetCorpse.Inventory.GetItems()
	if len(items) == 0 {
		player.Broadcast("The corpse has nothing to loot.")
		return
	}

	lootedItems := make([]string, 0)
	lootedItemDetails := make([]*components.Item, 0)
	for _, item := range items {
		if playerInventory.IsFull() {
			player.Broadcast("Your inventory is full!")
			break
		}

		if playerInventory.AddItem(item.Clone()) {
			lootedItems = append(lootedItems, item.Name)
			lootedItemDetails = append(lootedItemDetails, item)
		}
	}

	for _, item := range lootedItemDetails {
		targetCorpse.Inventory.RemoveItem(item.ID, item.Quantity)
	}

	if len(lootedItems) > 0 {
		player.Broadcast(fmt.Sprintf("You looted: %s", strings.Join(lootedItems, ", ")))
		player.Area.Broadcast(fmt.Sprintf("%s loots %s.", player.Name, targetCorpse.GetDescription()), player)

		// Mark corpse as looted so it decays in 5 seconds
		targetCorpse.MarkAsLooted()
	} else {
		player.Broadcast("You couldn't loot anything.")
	}
}

func (g *Game) handleLootAll(player *components.Player, game *Game) {
	// Find all corpses in the area
	corpses, err := g.world.FindEntitiesByComponentPredicate("Corpse", func(i interface{}) bool {
		c, ok := i.(*components.Corpse)
		return ok && c.Area == player.Area
	})

	if err != nil || len(corpses) == 0 {
		player.Broadcast("There are no corpses here to loot.")
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

	totalLootedItems := make([]string, 0)
	corpsesLooted := 0

	for _, corpseEntity := range corpses {
		corpseComp, err := g.world.GetComponent(corpseEntity.ID, "Corpse")
		if err != nil {
			continue
		}

		corpse := corpseComp.(*components.Corpse)

		// Skip corpses with no inventory or empty inventory
		if corpse.Inventory == nil {
			continue
		}

		items := corpse.Inventory.GetItems()
		if len(items) == 0 {
			continue
		}

		// Loot items from this corpse
		lootedFromCorpse := false
		lootedItemDetails := make([]*components.Item, 0)
		for _, item := range items {
			if playerInventory.IsFull() {
				player.Broadcast("Your inventory is full!")
				goto done
			}

			if playerInventory.AddItem(item.Clone()) {
				totalLootedItems = append(totalLootedItems, item.Name)
				lootedFromCorpse = true
				lootedItemDetails = append(lootedItemDetails, item)
			}
		}

		for _, item := range lootedItemDetails {
			corpse.Inventory.RemoveItem(item.ID, item.Quantity)
		}

		if lootedFromCorpse {
			corpsesLooted++
			// Mark corpse as looted so it decays in 5 seconds
			corpse.MarkAsLooted()
		}
	}

done:
	if len(totalLootedItems) > 0 {
		player.Broadcast(fmt.Sprintf("You looted %d corpse(s) and found: %s", corpsesLooted, strings.Join(totalLootedItems, ", ")))
		player.Area.Broadcast(fmt.Sprintf("%s loots all the corpses.", player.Name), player)
	} else {
		player.Broadcast("There was nothing to loot.")
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
	output.WriteString("==============================================\n")
	output.WriteString("                 INVENTORY                    \n")
	output.WriteString("==============================================\n\n")

	for _, item := range items {
		if item.Stackable && item.Quantity > 1 {
			output.WriteString(fmt.Sprintf("  %-30s x%d\n", item.Name, item.Quantity))
		} else {
			output.WriteString(fmt.Sprintf("  %s\n", item.Name))
		}
	}

	inventory.RLock()
	if inventory.MaxSlots > 0 {
		output.WriteString(fmt.Sprintf("\n(%d/%d slots used)\n", len(inventory.Items), inventory.MaxSlots))
	} else {
		totalItems := 0
		for _, item := range inventory.Items {
			if item == nil {
				continue
			}
			item.RLock()
			qty := item.Quantity
			stackable := item.Stackable
			item.RUnlock()
			if stackable && qty > 1 {
				totalItems += qty
			} else {
				totalItems += 1
			}
		}
		output.WriteString(fmt.Sprintf("\n(Bag of holding: %d items across %d stacks)\n", totalItems, len(inventory.Items)))
	}
	inventory.RUnlock()

	output.WriteString("==============================================\n")

	player.Broadcast(output.String())
}

func (g *Game) handleGet(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Get what? Usage: get <item> [quantity]")
		return
	}

	argName, quantity := parseArgsWithQuantity(args)
	itemName := strings.ToLower(argName)

	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}

	items := player.Area.GetItems()
	var targetItem *components.Item
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), itemName) {
			targetItem = item
			break
		}
	}

	if targetItem == nil {
		player.Broadcast("You don't see that here.")
		return
	}

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
	if inventory.IsFull() {
		player.Broadcast("Your inventory is full!")
		return
	}

	takeQuantity := targetItem.Quantity
	if quantity != -1 {
		if quantity > targetItem.Quantity {
			takeQuantity = targetItem.Quantity
		} else {
			takeQuantity = quantity
		}
	} else if !targetItem.Stackable {
		takeQuantity = 1
	}

	removed := player.Area.RemoveItem(targetItem.ID, takeQuantity)
	if removed == nil {
		player.Broadcast("You don't see that here.")
		return
	}

	if !inventory.AddItem(removed.Clone()) {
		player.Area.AddItem(removed)
		player.Broadcast("Your inventory is full!")
		return
	}

	if removed.Quantity > 1 {
		player.Broadcast(fmt.Sprintf("You pick up %s x%d.", removed.Name, removed.Quantity))
		player.Area.Broadcast(fmt.Sprintf("%s picks up %s x%d.", player.Name, removed.Name, removed.Quantity), player)
	} else {
		player.Broadcast(fmt.Sprintf("You pick up %s.", removed.Name))
		player.Area.Broadcast(fmt.Sprintf("%s picks up %s.", player.Name, removed.Name), player)
	}
}

func parseArgsWithQuantity(args []string) (string, int) {
	if len(args) == 0 {
		return "", 0
	}
	last := args[len(args)-1]
	if qty, err := strconv.Atoi(last); err == nil && qty > 0 {
		return strings.Join(args[:len(args)-1], " "), qty
	}
	return strings.Join(args, " "), -1 // -1 means "all"
}

func (g *Game) handleDrop(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Drop what? Usage: drop <item> [quantity]")
		return
	}

	itemName, quantity := parseArgsWithQuantity(args)
	matcher, _, isPattern, err := buildItemMatcher(itemName)
	if err != nil {
		player.Broadcast("Invalid pattern.")
		return
	}
	if strings.ContainsAny(itemName, "*?") || strings.ContainsAny(normalizePattern(itemName), "*?") {
		isPattern = true
	}

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

	dropped, droppedNames := dropMatchingItems(player, inventory, matcher, quantity)
	if dropped == 0 {
		if isPattern {
			player.Broadcast("No items matched that pattern.")
		} else {
			player.Broadcast("You don't have that item.")
		}
		return
	}
	player.Broadcast(fmt.Sprintf("You dropped %s.", strings.Join(droppedNames, ", ")))
	if player.Area != nil {
		player.Area.Broadcast(fmt.Sprintf("%s dropped some items.", player.Name), player)
	}
}

func (g *Game) handleDropAll(player *components.Player, args []string, game *Game) {
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

	matcher := func(name string) bool { return true }
	if len(args) > 0 {
		pattern := strings.TrimSpace(strings.Join(args, " "))
		parsed, _, _, err := buildItemMatcher(pattern)
		if err != nil {
			player.Broadcast("Invalid pattern.")
			return
		}
		matcher = parsed
	}

	dropped, droppedNames := dropMatchingItems(player, inventory, matcher, -1)
	if dropped == 0 {
		player.Broadcast("No items matched that pattern.")
		return
	}
	player.Broadcast(fmt.Sprintf("You dropped %s.", strings.Join(droppedNames, ", ")))
	if player.Area != nil {
		player.Area.Broadcast(fmt.Sprintf("%s dropped some items.", player.Name), player)
	}
}

func dropMatchingItems(player *components.Player, inventory *components.Inventory, matcher func(string) bool, limit int) (int, []string) {
	items := inventory.GetItems()
	if len(items) == 0 {
		return 0, nil
	}

	type dropSpec struct {
		name      string
		quantity  int
		stackable bool
	}

	byID := make(map[string]*dropSpec)

	// Collect items to drop
	remainingLimit := limit

	for _, item := range items {
		if item == nil {
			continue
		}
		if !itemMatches(matcher, item) {
			continue
		}

		toDrop := item.Quantity
		if remainingLimit != -1 {
			if remainingLimit <= 0 {
				break
			}
			if toDrop > remainingLimit {
				toDrop = remainingLimit
			}
		}

		spec, ok := byID[item.ID]
		if !ok {
			spec = &dropSpec{name: item.Name, stackable: item.Stackable}
			byID[item.ID] = spec
		}

		spec.quantity += toDrop

		if remainingLimit != -1 {
			remainingLimit -= toDrop
		}
	}

	if len(byID) == 0 {
		return 0, nil
	}

	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return 0, nil
	}

	var droppedNames []string
	droppedCount := 0
	for id, spec := range byID {
		if spec.quantity <= 0 {
			continue
		}

		if spec.stackable {
			removed := inventory.RemoveItem(id, spec.quantity)
			if removed != nil {
				removed.Quantity = spec.quantity
				player.Area.AddItem(removed)
				droppedCount++
				if spec.quantity > 1 {
					droppedNames = append(droppedNames, fmt.Sprintf("%s x%d", spec.name, spec.quantity))
				} else {
					droppedNames = append(droppedNames, spec.name)
				}
			}
			continue
		}

		for i := 0; i < spec.quantity; i++ {
			removed := inventory.RemoveItem(id, 1)
			if removed == nil {
				break
			}
			player.Area.AddItem(removed)
			droppedCount++
		}
		if spec.quantity > 1 {
			droppedNames = append(droppedNames, fmt.Sprintf("%s x%d", spec.name, spec.quantity))
		} else {
			droppedNames = append(droppedNames, spec.name)
		}
	}

	return droppedCount, droppedNames
}

func itemMatches(matcher func(string) bool, item *components.Item) bool {
	if matcher(item.Name) {
		return true
	}
	if item.ID == "" {
		return false
	}
	lookup := strings.ReplaceAll(item.ID, "_", " ")
	return matcher(lookup)
}

func (g *Game) handleSacrifice(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Sacrifice what? Usage: sacrifice <item> or sacrifice all")
		return
	}

	arg := strings.TrimSpace(strings.Join(args, " "))
	if strings.ToLower(arg) == "all" {
		g.handleSacrificeAll(player, nil, game)
		return
	}

	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}

	matcher, _, isPattern, err := buildItemMatcher(arg)
	if err != nil {
		player.Broadcast("Invalid pattern.")
		return
	}
	if strings.ContainsAny(arg, "*?") || strings.ContainsAny(normalizePattern(arg), "*?") {
		isPattern = true
	}

	sacrificedCount, sacrificedNames := sacrificeMatchingItems(player, matcher)

	if sacrificedCount == 0 {
		if isPattern {
			player.Broadcast("No items matched that pattern.")
		} else {
			player.Broadcast("You don't see that here.")
		}
		return
	}

	player.Broadcast(fmt.Sprintf("You sacrificed %s.", strings.Join(sacrificedNames, ", ")))
	player.Area.Broadcast(fmt.Sprintf("%s sacrificed %s to the void.", player.Name, strings.Join(sacrificedNames, ", ")), player)
}

func (g *Game) handleSacrificeAll(player *components.Player, args []string, game *Game) {
	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}

	matcher := func(name string) bool { return true }
	if len(args) > 0 {
		pattern := strings.TrimSpace(strings.Join(args, " "))
		parsed, _, _, err := buildItemMatcher(pattern)
		if err != nil {
			player.Broadcast("Invalid pattern.")
			return
		}
		matcher = parsed
	}

	sacrificedCount, sacrificedNames := sacrificeMatchingItems(player, matcher)

	if sacrificedCount == 0 {
		player.Broadcast("There is nothing here to sacrifice.")
		return
	}

	player.Broadcast(fmt.Sprintf("You sacrificed %s.", strings.Join(sacrificedNames, ", ")))
	player.Area.Broadcast(fmt.Sprintf("%s sacrificed everything to the void.", player.Name), player)
}

func sacrificeMatchingItems(player *components.Player, matcher func(string) bool) (int, []string) {
	items := player.Area.GetItems()
	if len(items) == 0 {
		return 0, nil
	}

	type sacReport struct {
		name     string
		quantity int
	}
	reports := make(map[string]*sacReport)
	sacrificedCount := 0

	for _, item := range items {
		if !itemMatches(matcher, item) {
			continue
		}

		removed := player.Area.RemoveItem(item.ID, item.Quantity)
		if removed != nil {
			sacrificedCount++
			if rep, ok := reports[item.ID]; ok {
				rep.quantity += removed.Quantity
			} else {
				reports[item.ID] = &sacReport{name: removed.Name, quantity: removed.Quantity}
			}
		}
	}

	if sacrificedCount == 0 {
		return 0, nil
	}

	var sacrificedNames []string
	for _, rep := range reports {
		if rep.quantity > 1 {
			sacrificedNames = append(sacrificedNames, fmt.Sprintf("%s x%d", rep.name, rep.quantity))
		} else {
			sacrificedNames = append(sacrificedNames, rep.name)
		}
	}

	return sacrificedCount, sacrificedNames
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
