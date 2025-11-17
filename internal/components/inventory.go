package components

import "sync"

type Inventory struct {
	sync.RWMutex

	Items    []*Item
	MaxSlots int // 0 = unlimited
}

func NewInventory(maxSlots int) *Inventory {
	return &Inventory{
		Items:    make([]*Item, 0),
		MaxSlots: maxSlots,
	}
}

func (inv *Inventory) Type() string {
	return "Inventory"
}

func (inv *Inventory) AddItem(item *Item) bool {
	inv.Lock()
	defer inv.Unlock()

	if inv.MaxSlots > 0 && len(inv.Items) >= inv.MaxSlots {
		return false
	}

	// Check if item is stackable and already exists
	if item.Stackable {
		for _, existing := range inv.Items {
			existing.Lock()
			if existing.ID == item.ID {
				existing.Quantity += item.Quantity
				existing.Unlock()
				return true
			}
			existing.Unlock()
		}
	}

	// Add as new item
	inv.Items = append(inv.Items, item)
	return true
}

func (inv *Inventory) RemoveItem(itemID string, quantity int) *Item {
	inv.Lock()
	defer inv.Unlock()

	for i, item := range inv.Items {
		item.Lock()
		if item.ID == itemID {
			if item.Stackable && item.Quantity > quantity {
				// Partial removal from stack
				item.Quantity -= quantity
				removed := item.Clone()
				removed.Quantity = quantity
				item.Unlock()
				return removed
			} else {
				// Remove entire item/stack
				removed := item.Clone()
				item.Unlock()
				inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
				return removed
			}
		}
		item.Unlock()
	}

	return nil
}

func (inv *Inventory) FindItem(itemID string) *Item {
	inv.RLock()
	defer inv.RUnlock()

	for _, item := range inv.Items {
		item.RLock()
		if item.ID == itemID {
			clone := item.Clone()
			item.RUnlock()
			return clone
		}
		item.RUnlock()
	}

	return nil
}

func (inv *Inventory) GetItems() []*Item {
	inv.RLock()
	defer inv.RUnlock()

	items := make([]*Item, len(inv.Items))
	for i, item := range inv.Items {
		items[i] = item.Clone()
	}
	return items
}

func (inv *Inventory) IsFull() bool {
	inv.RLock()
	defer inv.RUnlock()

	return inv.MaxSlots > 0 && len(inv.Items) >= inv.MaxSlots
}
