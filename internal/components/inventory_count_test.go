package components

import "testing"

func TestInventoryCountItem(t *testing.T) {
	inv := NewInventory(0)
	inv.AddItem(&Item{ID: "gold_coin", Name: "Gold Coin", Stackable: true, Quantity: 10})
	inv.AddItem(&Item{ID: "gold_coin", Name: "Gold Coin", Stackable: true, Quantity: 5})
	inv.AddItem(&Item{ID: "bread", Name: "Bread", Stackable: true, Quantity: 2})

	if got := inv.CountItem("gold_coin"); got != 15 {
		t.Errorf("gold count = %d, want 15 (stacked)", got)
	}
	if got := inv.CountItem("bread"); got != 2 {
		t.Errorf("bread count = %d, want 2", got)
	}
	if got := inv.CountItem("nonexistent"); got != 0 {
		t.Errorf("missing item count = %d, want 0", got)
	}
}
