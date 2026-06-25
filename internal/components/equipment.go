package components

import (
	"fmt"
	"sort"
	"sync"
)

// Equipment is what a creature is wearing/wielding, one item per slot. It
// composes with the stat engine: the weapon adds swing damage, worn gear soaks
// damage (Armor) and raises the HP pool (HPBonus).
type Equipment struct {
	sync.RWMutex
	Slots map[EquipSlot]*Item
}

func NewEquipment() *Equipment {
	return &Equipment{Slots: make(map[EquipSlot]*Item)}
}

func (e *Equipment) Type() string { return "Equipment" }

// Equip places item in its slot and returns whatever it displaced (or nil).
func (e *Equipment) Equip(item *Item) *Item {
	e.Lock()
	defer e.Unlock()
	prev := e.Slots[item.Slot]
	e.Slots[item.Slot] = item
	return prev
}

// Remove clears a slot and returns the item that was there (or nil).
func (e *Equipment) Remove(slot EquipSlot) *Item {
	e.Lock()
	defer e.Unlock()
	it := e.Slots[slot]
	delete(e.Slots, slot)
	return it
}

func (e *Equipment) InSlot(slot EquipSlot) *Item {
	e.RLock()
	defer e.RUnlock()
	return e.Slots[slot]
}

// WeaponDamage is the equipped weapon's added swing range (0,0 if unarmed).
func (e *Equipment) WeaponDamage() (min, max int) {
	e.RLock()
	defer e.RUnlock()
	if w := e.Slots[SlotWeapon]; w != nil {
		return w.DamageMin, w.DamageMax
	}
	return 0, 0
}

// ArmorValue is the total flat damage reduction from all worn pieces.
func (e *Equipment) ArmorValue() int {
	e.RLock()
	defer e.RUnlock()
	total := 0
	for _, it := range e.Slots {
		if it != nil {
			total += it.Armor
		}
	}
	return total
}

// HPBonus is the total max-HP boost from all worn pieces.
func (e *Equipment) HPBonus() int {
	e.RLock()
	defer e.RUnlock()
	total := 0
	for _, it := range e.Slots {
		if it != nil {
			total += it.HPBonus
		}
	}
	return total
}

// EquipView is the client-facing snapshot of one worn item.
type EquipView struct {
	Slot    string `json:"slot"`
	Name    string `json:"name"`
	Damage  string `json:"damage,omitempty"`
	Armor   int    `json:"armor,omitempty"`
	HPBonus int    `json:"hp_bonus,omitempty"`
}

// Snapshot returns worn items in a stable slot order for an equipment event.
func (e *Equipment) Snapshot() []EquipView {
	e.RLock()
	defer e.RUnlock()
	var out []EquipView
	for _, it := range e.Slots {
		if it == nil {
			continue
		}
		v := EquipView{Slot: it.Slot.String(), Name: it.Name, Armor: it.Armor, HPBonus: it.HPBonus}
		if it.DamageMax > 0 {
			v.Damage = fmt.Sprintf("%d-%d", it.DamageMin, it.DamageMax)
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out
}
