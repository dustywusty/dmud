package components

import "testing"

func TestEquipmentEquipDisplaces(t *testing.T) {
	eq := NewEquipment()
	sword := &Item{ID: "sword", Name: "Iron Sword", Slot: SlotWeapon, DamageMin: 3, DamageMax: 6}
	axe := &Item{ID: "axe", Name: "Battle Axe", Slot: SlotWeapon, DamageMin: 5, DamageMax: 9}

	if prev := eq.Equip(sword); prev != nil {
		t.Fatalf("first equip should displace nothing, got %v", prev)
	}
	prev := eq.Equip(axe)
	if prev == nil || prev.ID != "sword" {
		t.Fatalf("equipping a second weapon should displace the sword, got %v", prev)
	}
	if w := eq.InSlot(SlotWeapon); w == nil || w.ID != "axe" {
		t.Errorf("weapon slot should now hold the axe")
	}
}

func TestEquipmentDamageArmorHP(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(&Item{ID: "axe", Name: "Axe", Slot: SlotWeapon, DamageMin: 5, DamageMax: 9})
	eq.Equip(&Item{ID: "mail", Name: "Chainmail", Slot: SlotArmor, Armor: 3, HPBonus: 20})
	eq.Equip(&Item{ID: "shield", Name: "Buckler", Slot: SlotShield, Armor: 2})

	if lo, hi := eq.WeaponDamage(); lo != 5 || hi != 9 {
		t.Errorf("weapon damage = %d-%d, want 5-9", lo, hi)
	}
	if a := eq.ArmorValue(); a != 5 {
		t.Errorf("armor = %d, want 5 (3+2)", a)
	}
	if hp := eq.HPBonus(); hp != 20 {
		t.Errorf("hp bonus = %d, want 20", hp)
	}
}

func TestEquipmentRemove(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(&Item{ID: "axe", Name: "Axe", Slot: SlotWeapon, DamageMin: 5, DamageMax: 9})

	if got := eq.Remove(SlotWeapon); got == nil || got.ID != "axe" {
		t.Fatalf("remove should return the axe, got %v", got)
	}
	if eq.InSlot(SlotWeapon) != nil {
		t.Errorf("slot should be empty after remove")
	}
	if lo, hi := eq.WeaponDamage(); lo != 0 || hi != 0 {
		t.Errorf("unarmed weapon damage should be 0-0, got %d-%d", lo, hi)
	}
}

func TestEquipmentSnapshotStableAndSorted(t *testing.T) {
	eq := NewEquipment()
	eq.Equip(&Item{ID: "axe", Name: "Axe", Slot: SlotWeapon, DamageMin: 5, DamageMax: 9})
	eq.Equip(&Item{ID: "mail", Name: "Mail", Slot: SlotArmor, Armor: 3, HPBonus: 20})

	s := eq.Snapshot()
	if len(s) != 2 {
		t.Fatalf("want 2 worn items, got %d", len(s))
	}
	// Sorted by slot string: "armor" < "weapon".
	if s[0].Slot != "armor" || s[1].Slot != "weapon" {
		t.Errorf("snapshot not sorted by slot: %+v", s)
	}
	if s[1].Damage != "5-9" {
		t.Errorf("weapon damage string = %q, want 5-9", s[1].Damage)
	}
}

func TestEquipSlotFromName(t *testing.T) {
	cases := map[string]EquipSlot{
		"weapon": SlotWeapon, "wield": SlotWeapon,
		"armor": SlotArmor, "body": SlotArmor,
		"shield": SlotShield, "offhand": SlotShield,
		"banana": SlotNone,
	}
	for in, want := range cases {
		if got := EquipSlotFromName(in); got != want {
			t.Errorf("EquipSlotFromName(%q) = %d, want %d", in, got, want)
		}
	}
}
