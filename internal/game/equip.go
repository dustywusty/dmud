package game

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
	"encoding/json"
	"fmt"
	"strings"
)

func (g *Game) getOrCreateEquipment(entityID common.EntityID) (*components.Equipment, error) {
	eq, err := ecs.GetTypedComponent[*components.Equipment](g.world, entityID, "Equipment")
	if err == nil && eq != nil {
		return eq, nil
	}
	entity, err := g.world.FindEntity(entityID)
	if err != nil {
		return nil, err
	}
	eq = components.NewEquipment()
	g.world.AddComponent(&entity, eq)
	return eq, nil
}

// handleEquip wields/wears an equippable item from the player's inventory,
// returning any displaced gear to the bag.
func (g *Game) handleEquip(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Equip what? Usage: equip <item> (also: wield, wear)")
		return
	}
	name := strings.ToLower(strings.Join(args, " "))

	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}
	invComp, err := g.world.GetComponent(entityID, "Inventory")
	if err != nil {
		player.Broadcast("You have no inventory.")
		return
	}
	inv := invComp.(*components.Inventory)

	var match *components.Item
	for _, it := range inv.GetItems() {
		if strings.Contains(strings.ToLower(it.Name), name) {
			match = it
			break
		}
	}
	if match == nil {
		player.Broadcast("You aren't carrying that.")
		return
	}
	if !match.Equippable() {
		player.Broadcast(fmt.Sprintf("You can't equip %s.", match.Name))
		return
	}

	removed := inv.RemoveItem(match.ID, 1)
	if removed == nil {
		player.Broadcast("You aren't carrying that.")
		return
	}
	removed.Quantity = 1

	eq, err := g.getOrCreateEquipment(entityID)
	if err != nil {
		inv.AddItem(removed)
		return
	}
	if prev := eq.Equip(removed); prev != nil {
		inv.AddItem(prev) // displaced gear goes back to the bag
		player.Broadcast(fmt.Sprintf("You remove %s.", prev.Name))
	}

	verb := "equip"
	switch removed.Slot {
	case components.SlotWeapon:
		verb = "wield"
	case components.SlotArmor, components.SlotShield:
		verb = "wear"
	}
	player.Broadcast(fmt.Sprintf("You %s %s.", verb, removed.Name))
	if player.Area != nil {
		player.Area.Broadcast(fmt.Sprintf("%s %ss %s.", player.Name, verb, removed.Name), player)
	}
	g.broadcastEquipment(player, entityID)
	player.BroadcastState(g.world.AsWorldLike(), entityID)
}

// handleRemove takes off a worn item (by name or slot word) back into the bag.
func (g *Game) handleRemove(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Remove what? Usage: remove <item|weapon|armor|shield>")
		return
	}
	name := strings.ToLower(strings.Join(args, " "))

	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}
	eq, err := ecs.GetTypedComponent[*components.Equipment](g.world, entityID, "Equipment")
	if err != nil || eq == nil {
		player.Broadcast("You have nothing equipped.")
		return
	}

	slot := components.EquipSlotFromName(name)
	if slot == components.SlotNone {
		for _, v := range eq.Snapshot() {
			if strings.Contains(strings.ToLower(v.Name), name) {
				slot = components.EquipSlotFromName(v.Slot)
				break
			}
		}
	}
	if slot == components.SlotNone {
		player.Broadcast("You don't have that equipped.")
		return
	}
	it := eq.Remove(slot)
	if it == nil {
		player.Broadcast("You don't have that equipped.")
		return
	}
	if invComp, err := g.world.GetComponent(entityID, "Inventory"); err == nil {
		invComp.(*components.Inventory).AddItem(it)
	}
	player.Broadcast(fmt.Sprintf("You remove %s.", it.Name))
	g.broadcastEquipment(player, entityID)
	player.BroadcastState(g.world.AsWorldLike(), entityID)
}

// handleEquipment lists what the player has equipped.
func (g *Game) handleEquipment(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}
	eq, err := ecs.GetTypedComponent[*components.Equipment](g.world, entityID, "Equipment")
	if err != nil || eq == nil {
		player.Broadcast("You have nothing equipped.")
		return
	}
	worn := eq.Snapshot()
	if len(worn) == 0 {
		player.Broadcast("You have nothing equipped.")
		return
	}
	var b strings.Builder
	b.WriteString("You are equipped with:\n")
	for _, v := range worn {
		b.WriteString(fmt.Sprintf("  [%s] %s", v.Slot, v.Name))
		var extras []string
		if v.Damage != "" {
			extras = append(extras, "dmg "+v.Damage)
		}
		if v.Armor > 0 {
			extras = append(extras, fmt.Sprintf("armor %d", v.Armor))
		}
		if v.HPBonus > 0 {
			extras = append(extras, fmt.Sprintf("+%d HP", v.HPBonus))
		}
		if len(extras) > 0 {
			b.WriteString(" (" + strings.Join(extras, ", ") + ")")
		}
		b.WriteString("\n")
	}
	player.Broadcast(b.String())
	g.broadcastEquipment(player, entityID)
}

// broadcastEquipment pushes the structured equipment event for tag clients.
func (g *Game) broadcastEquipment(player *components.Player, entityID common.EntityID) {
	eq, err := ecs.GetTypedComponent[*components.Equipment](g.world, entityID, "Equipment")
	if err != nil || eq == nil {
		return
	}
	payload := struct {
		Type  string                 `json:"type"`
		Slots []components.EquipView `json:"slots"`
	}{"equipment", eq.Snapshot()}
	if data, err := json.Marshal(payload); err == nil {
		player.Broadcast(util.TagMessage("EVENT", string(data)))
	}
}
