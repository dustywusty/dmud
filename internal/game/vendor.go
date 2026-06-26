package game

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const goldItemID = "gold_coin"

// handleGive lets a player hand an item to an NPC. For a vendor that wants a
// particular item (Wylie wants cookies), this is how reputation is earned.
func (g *Game) handleGive(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Give what to whom? Usage: give <item> [qty] to <npc>")
		return
	}
	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}

	itemTokens, npcTokens := splitOnTo(args)
	if len(npcTokens) == 0 {
		player.Broadcast("Give to whom? Usage: give <item> [qty] to <npc>")
		return
	}

	itemName, qty := parseArgsWithQuantity(itemTokens)
	if strings.TrimSpace(itemName) == "" {
		player.Broadcast("Give what? Usage: give <item> [qty] to <npc>")
		return
	}
	npcName := strings.ToLower(strings.Join(npcTokens, " "))

	npc := g.findAreaNPC(player, npcName)
	if npc == nil {
		player.Broadcast("You don't see them here.")
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

	held := findInventoryItemByName(inventory, itemName)
	if held == nil {
		player.Broadcast("You don't have that.")
		return
	}

	// How many to give: all of that item by default, or the requested amount.
	giveQty := held.Quantity
	if qty != -1 && qty < giveQty {
		giveQty = qty
	}

	vendor, isVendor := components.VendorForNPC(npc.TemplateID)
	wantsThis := isVendor && vendor.DesiredItemID != "" && held.ID == vendor.DesiredItemID && vendor.FactionID != ""

	npc.HoldConversation(60 * time.Second)

	if !wantsThis {
		// Not a vendor faction-gift. Is the item a turn-in for a quest this NPC
		// offers? If so, `give` completes (or nudges) the quest — the intuitive
		// verb players reach for.
		handler := &components.QuestDialogueHandler{World: g.world.AsWorldLike()}
		if handler.TryTurnInItem(player, playerEntity, npc, held.ID) {
			return
		}

		// NPC declines; the item stays in the player's inventory.
		if isVendor && vendor.GiftRejectLine != "" {
			player.Broadcast(fmt.Sprintf("%s says: %s", npc.Name, vendor.GiftRejectLine))
		} else {
			player.Broadcast(fmt.Sprintf("%s doesn't seem interested in that.", npc.Name))
		}
		return
	}

	removed := inventory.RemoveItem(held.ID, giveQty)
	if removed == nil {
		player.Broadcast("You don't have that.")
		return
	}

	label := itemStackLabel(removed.Name, removed.Quantity)
	player.Broadcast(fmt.Sprintf("You give %s to %s.", label, npc.Name))
	player.Area.Broadcast(fmt.Sprintf("%s gives %s to %s.", player.Name, label, npc.Name), player)
	if vendor.GiftAcceptLine != "" {
		player.Broadcast(fmt.Sprintf("%s says: %s", npc.Name, vendor.GiftAcceptLine))
	}

	factions := g.getFactions(playerEntity)
	prevRep := factions.Get(vendor.FactionID)
	gained := vendor.RepPerItem * removed.Quantity
	newRep := factions.Add(vendor.FactionID, gained)
	reportStandingChange(player, npc, vendor, prevRep, newRep, gained)
}

// handleBuy purchases a ware from a vendor NPC in the player's area. Purchases
// are gated behind faction standing.
func (g *Game) handleBuy(player *components.Player, args []string, game *Game) {
	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}

	npc, vendor := g.findAreaVendor(player, "")
	if npc == nil {
		player.Broadcast("There's no merchant here to buy from.")
		return
	}

	if len(args) == 0 {
		player.Broadcast("Buy what? Usage: buy <item> [qty] (try 'list' to see the wares)")
		return
	}

	itemName, qty := parseArgsWithQuantity(args)
	if qty < 1 {
		qty = 1
	}

	ware, tmpl, ok := matchWare(vendor, itemName)
	if !ok {
		player.Broadcast(fmt.Sprintf("%s says: I don't sell that, love. Try 'list' to see what I've got.", npc.Name))
		return
	}

	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}

	// Faction gate (the vendor baseline, raised by pricier wares).
	requiredRep := vendor.MinRepToBuy
	if ware.MinRep > requiredRep {
		requiredRep = ware.MinRep
	}
	factions := g.getFactions(playerEntity)
	rep := factions.Get(vendor.FactionID)
	if rep < requiredRep {
		needTitle := "a regular"
		if def := components.FactionRegistry[vendor.FactionID]; def != nil {
			needTitle = def.RankTitle(requiredRep)
		}
		player.Broadcast(fmt.Sprintf("%s says: I'd love to sell you that, but I only let %s and above take this one. Bring me cookies, yeah?", npc.Name, needTitle))
		return
	}

	invComp, err := g.world.GetComponent(playerEntity, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory!")
		return
	}
	inventory := invComp.(*components.Inventory)

	totalPrice := ware.Price * qty
	gold := countInventoryItem(inventory, goldItemID)
	if gold < totalPrice {
		player.Broadcast(fmt.Sprintf("%s says: That'll be %d gold for %d, and you've only got %d. Maybe next time!", npc.Name, totalPrice, qty, gold))
		return
	}

	// Take payment, then hand over the goods.
	inventory.RemoveItem(goldItemID, totalPrice)
	for i := 0; i < qty; i++ {
		inventory.AddItem(components.CreateItem(ware.ItemID, 1))
	}

	label := itemStackLabel(tmpl.Name, qty)
	player.Broadcast(fmt.Sprintf("You buy %s from %s for %d gold.", label, npc.Name, totalPrice))
	player.Area.Broadcast(fmt.Sprintf("%s buys %s from %s.", player.Name, label, npc.Name), player)
}

// sellPrice is what a vendor pays for an item: half its value, at least 1.
func sellPrice(value int) int {
	if p := value / 2; p > 1 {
		return p
	}
	return 1
}

// handleSell sells an item from the player's inventory to a vendor here for gold.
func (g *Game) handleSell(player *components.Player, args []string, game *Game) {
	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}
	npc, _ := g.findAreaVendor(player, "")
	if npc == nil {
		player.Broadcast("There's no merchant here to sell to.")
		return
	}
	if len(args) == 0 {
		player.Broadcast("Sell what? Usage: sell <item> [qty]")
		return
	}

	itemName, qty := parseArgsWithQuantity(args)
	if qty < 1 {
		qty = 1
	}

	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}
	invComp, err := g.world.GetComponent(playerEntity, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory!")
		return
	}
	inventory := invComp.(*components.Inventory)

	// Find a matching item to sell (never the gold itself).
	var match *components.Item
	for _, it := range inventory.GetItems() {
		if it.ID == goldItemID {
			continue
		}
		if strings.Contains(strings.ToLower(it.Name), strings.ToLower(itemName)) {
			match = it
			break
		}
	}
	if match == nil {
		player.Broadcast(fmt.Sprintf("%s says: You're not carrying that, love.", npc.Name))
		return
	}
	if match.Value <= 0 {
		player.Broadcast(fmt.Sprintf("%s says: That's worthless to me, I'm afraid.", npc.Name))
		return
	}

	removed := inventory.RemoveItem(match.ID, qty)
	if removed == nil {
		player.Broadcast(fmt.Sprintf("%s says: You're not carrying that, love.", npc.Name))
		return
	}
	total := sellPrice(match.Value) * removed.Quantity
	if total < 1 {
		total = 1
	}
	inventory.AddItem(components.CreateItem(goldItemID, total))

	label := itemStackLabel(match.Name, removed.Quantity)
	player.Broadcast(fmt.Sprintf("You sell %s to %s for %d gold.", label, npc.Name, total))
	player.Area.Broadcast(fmt.Sprintf("%s sells %s to %s.", player.Name, label, npc.Name), player)
	player.BroadcastState(g.world.AsWorldLike(), playerEntity)
}

// handleGold reports the player's current coin purse.
func (g *Game) handleGold(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}
	gold := 0
	if invComp, err := g.world.GetComponent(entityID, "Inventory"); err == nil {
		gold = invComp.(*components.Inventory).CountItem(goldItemID)
	}
	player.Broadcast(fmt.Sprintf("You have %d gold.", gold))
}

// handleList shows a vendor's wares along with the player's current standing.
func (g *Game) handleList(player *components.Player, args []string, game *Game) {
	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}

	npc, vendor := g.findAreaVendor(player, "")
	if npc == nil {
		player.Broadcast("There's no merchant here.")
		return
	}

	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}
	rep := g.getFactions(playerEntity).Get(vendor.FactionID)
	canBuy := rep >= vendor.MinRepToBuy

	var b strings.Builder
	shopName := vendor.ShopName
	if shopName == "" {
		shopName = npc.Name + "'s wares"
	}
	b.WriteString(fmt.Sprintf("~~~ %s ~~~\n", shopName))

	if def := components.FactionRegistry[vendor.FactionID]; def != nil {
		b.WriteString(fmt.Sprintf("Your standing: %s (%d) with %s\n", def.RankTitle(rep), rep, def.Name))
		if canBuy {
			b.WriteString("You're welcome to buy anything here.\n")
		} else {
			b.WriteString(fmt.Sprintf("You must be %s to buy here -- bring cookies to earn it!\n", def.RankTitle(vendor.MinRepToBuy)))
		}
	}
	b.WriteString("\n")

	def := components.FactionRegistry[vendor.FactionID]
	for _, ware := range vendor.Wares {
		tmpl, ok := components.ItemTemplates[ware.ItemID]
		if !ok {
			continue
		}
		line := fmt.Sprintf("  %-24s %4d gold", tmpl.Name, ware.Price)
		if def != nil && ware.MinRep > vendor.MinRepToBuy {
			line += fmt.Sprintf("   (requires %s)", def.RankTitle(ware.MinRep))
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\nType 'buy <item>' to purchase. Mount a horse with 'mount', then 'ride <direction>' to gallop.")
	player.Broadcast(b.String())
}

// handleFaction shows the player's standing with every faction they've earned
// reputation with.
func (g *Game) handleFaction(player *components.Player, args []string, game *Game) {
	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}

	rep := g.getFactions(playerEntity).Snapshot()
	if len(rep) == 0 {
		player.Broadcast("You haven't earned standing with anyone yet. (Hint: Wylie loves cookies.)")
		return
	}

	var b strings.Builder
	b.WriteString("=============== STANDINGS ===============\n")
	for factionID, points := range rep {
		def := components.FactionRegistry[factionID]
		if def == nil {
			continue
		}
		b.WriteString(fmt.Sprintf("%s: %s (%d)\n", def.Name, def.RankTitle(points), points))
		if next, ok := def.NextRank(points); ok {
			b.WriteString(fmt.Sprintf("  %d more to reach %s\n", next.MinRep-points, next.Title))
		} else {
			b.WriteString("  (highest standing reached)\n")
		}
	}
	b.WriteString("========================================")
	player.Broadcast(b.String())
}

// reportStandingChange tells the player how a gift moved their reputation, and
// calls out rank-ups and crossing a vendor's buying threshold.
func reportStandingChange(player *components.Player, npc *components.NPC, vendor *components.VendorDef, prevRep, newRep, gained int) {
	def := components.FactionRegistry[vendor.FactionID]
	if def == nil {
		return
	}

	player.Broadcast(fmt.Sprintf("Your standing with %s rises by %d (now %d).", def.Name, gained, newRep))

	oldTitle := def.RankTitle(prevRep)
	newTitle := def.RankTitle(newRep)
	if newTitle != oldTitle {
		player.Broadcast(fmt.Sprintf("You are now regarded as %s by %s!", newTitle, def.Name))
	}

	switch {
	case prevRep < vendor.MinRepToBuy && newRep >= vendor.MinRepToBuy:
		player.Broadcast(fmt.Sprintf("%s says: That's it -- you're one of us now! Have a look at my wares any time. (try 'list')", npc.Name))
	case newRep < vendor.MinRepToBuy:
		player.Broadcast(fmt.Sprintf("%s says: %d more to go before I open the good shelves for you!", npc.Name, vendor.MinRepToBuy-newRep))
	}
}

// --- helpers ---

// splitOnTo splits "cookies to wylie" into item tokens and NPC tokens. It
// accepts an explicit "to" separator, and falls back to treating the last word
// as the NPC name when "to" is omitted ("give cookies wylie").
func splitOnTo(args []string) (itemTokens, npcTokens []string) {
	for i, a := range args {
		if strings.EqualFold(a, "to") {
			return args[:i], args[i+1:]
		}
	}
	if len(args) >= 2 {
		return args[:len(args)-1], args[len(args)-1:]
	}
	return args, nil
}

func (g *Game) findAreaNPC(player *components.Player, name string) *components.NPC {
	if player.Area == nil {
		return nil
	}
	for _, npc := range player.Area.GetNPCs(g.world.AsWorldLike()) {
		if strings.Contains(strings.ToLower(npc.Name), name) {
			return npc
		}
	}
	return nil
}

// findAreaVendor returns the first vendor NPC in the player's area (optionally
// matching a name) along with its vendor definition.
func (g *Game) findAreaVendor(player *components.Player, name string) (*components.NPC, *components.VendorDef) {
	if player.Area == nil {
		return nil, nil
	}
	for _, npc := range player.Area.GetNPCs(g.world.AsWorldLike()) {
		if name != "" && !strings.Contains(strings.ToLower(npc.Name), name) {
			continue
		}
		if vendor, ok := components.VendorForNPC(npc.TemplateID); ok {
			return npc, vendor
		}
	}
	return nil, nil
}

func (g *Game) getFactions(entityID common.EntityID) *components.Factions {
	if comp, err := g.world.GetComponent(entityID, "Factions"); err == nil {
		if factions, ok := comp.(*components.Factions); ok {
			return factions
		}
	}
	// Defensive: create one if it's somehow missing.
	factions := components.NewFactions()
	if entity, err := g.world.FindEntity(entityID); err == nil {
		g.world.AddComponent(&entity, factions)
	}
	return factions
}

// findInventoryItemByName returns a clone of the first inventory item whose name
// or ID contains the given text.
func findInventoryItemByName(inv *components.Inventory, name string) *components.Item {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, item := range inv.GetItems() {
		if strings.Contains(strings.ToLower(item.Name), name) {
			return item
		}
		if item.ID != "" && strings.Contains(strings.ReplaceAll(item.ID, "_", " "), name) {
			return item
		}
	}
	return nil
}

func countInventoryItem(inv *components.Inventory, itemID string) int {
	found := inv.FindItem(itemID)
	if found == nil {
		return 0
	}
	return found.Quantity
}

// matchWare finds the ware a typed name refers to, returning the ware and its
// item template.
func matchWare(vendor *components.VendorDef, typed string) (components.Ware, *components.Item, bool) {
	typed = strings.ToLower(strings.TrimSpace(typed))
	if typed == "" {
		return components.Ware{}, nil, false
	}
	for _, ware := range vendor.Wares {
		tmpl, ok := components.ItemTemplates[ware.ItemID]
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(tmpl.Name), typed) ||
			strings.Contains(strings.ReplaceAll(ware.ItemID, "_", " "), typed) {
			return ware, tmpl, true
		}
	}
	return components.Ware{}, nil, false
}

func itemStackLabel(name string, qty int) string {
	if qty > 1 {
		return fmt.Sprintf("%s x%d", name, qty)
	}
	return name
}
