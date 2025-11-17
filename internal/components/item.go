package components

import "sync"

type ItemType int

const (
	ItemTypeWeapon ItemType = iota
	ItemTypeArmor
	ItemTypeConsumable
	ItemTypeMisc
)

type Item struct {
	sync.RWMutex

	ID          string
	Name        string
	Description string
	Type        ItemType
	Value       int
	Stackable   bool
	Quantity    int
}

func (i *Item) Clone() *Item {
	i.RLock()
	defer i.RUnlock()

	return &Item{
		ID:          i.ID,
		Name:        i.Name,
		Description: i.Description,
		Type:        i.Type,
		Value:       i.Value,
		Stackable:   i.Stackable,
		Quantity:    i.Quantity,
	}
}
