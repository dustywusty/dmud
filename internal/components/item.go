package components

import "sync"

type ItemType int

const (
	ItemTypeWeapon ItemType = iota
	ItemTypeArmor
	ItemTypeConsumable
	ItemTypeMisc
)

// EquipSlot is where a piece of gear is worn. SlotNone means the item can't be
// equipped (consumables, junk, quest items).
type EquipSlot int

const (
	SlotNone EquipSlot = iota
	SlotWeapon
	SlotArmor
	SlotShield
)

func (s EquipSlot) String() string {
	switch s {
	case SlotWeapon:
		return "weapon"
	case SlotArmor:
		return "armor"
	case SlotShield:
		return "shield"
	default:
		return ""
	}
}

// EquipSlotFromName maps a user-typed slot word to a slot (SlotNone if unknown).
func EquipSlotFromName(name string) EquipSlot {
	switch name {
	case "weapon", "wield":
		return SlotWeapon
	case "armor", "armour", "body":
		return SlotArmor
	case "shield", "offhand":
		return SlotShield
	default:
		return SlotNone
	}
}

type Item struct {
	sync.RWMutex

	ID          string
	Name        string
	Description string
	Type        ItemType
	Value       int
	Stackable   bool
	Quantity    int

	// Equipment: when Slot != SlotNone the item can be worn/wielded. A weapon adds
	// DamageMin..DamageMax to a swing; worn gear soaks Armor damage and adds HPBonus
	// to the max HP pool.
	Slot      EquipSlot
	DamageMin int
	DamageMax int
	Armor     int
	HPBonus   int
}

// cloneLocked copies every field; callers must already hold a (read or write)
// lock on i. Keeping the field list in one place stops copies from drifting as
// the struct grows.
func (i *Item) cloneLocked() *Item {
	return &Item{
		ID:          i.ID,
		Name:        i.Name,
		Description: i.Description,
		Type:        i.Type,
		Value:       i.Value,
		Stackable:   i.Stackable,
		Quantity:    i.Quantity,
		Slot:        i.Slot,
		DamageMin:   i.DamageMin,
		DamageMax:   i.DamageMax,
		Armor:       i.Armor,
		HPBonus:     i.HPBonus,
	}
}

func (i *Item) Clone() *Item {
	i.RLock()
	defer i.RUnlock()
	return i.cloneLocked()
}

// Equippable reports whether the item can be worn or wielded.
func (i *Item) Equippable() bool {
	return i.Slot != SlotNone
}
