package components

// ItemTemplates defines all available items in the game
var ItemTemplates = map[string]Item{
	"rat_fur": {
		ID:          "rat_fur",
		Name:        "Rat Fur",
		Description: "A small patch of matted rat fur.",
		Type:        ItemTypeMisc,
		Value:       1,
		Stackable:   true,
		Quantity:    1,
	},
	"rat_tail": {
		ID:          "rat_tail",
		Name:        "Rat Tail",
		Description: "A long, scaly rat tail.",
		Type:        ItemTypeMisc,
		Value:       2,
		Stackable:   true,
		Quantity:    1,
	},
	"goblin_ear": {
		ID:          "goblin_ear",
		Name:        "Goblin Ear",
		Description: "A pointed goblin ear, still slightly warm.",
		Type:        ItemTypeMisc,
		Value:       5,
		Stackable:   true,
		Quantity:    1,
	},
	"rusty_dagger": {
		ID:          "rusty_dagger",
		Name:        "Rusty Dagger",
		Description: "A crude, rusty dagger that's seen better days.",
		Type:        ItemTypeWeapon,
		Value:       10,
		Stackable:   false,
		Quantity:    1,
	},
	"gold_coin": {
		ID:          "gold_coin",
		Name:        "Gold Coin",
		Description: "A shiny gold coin.",
		Type:        ItemTypeMisc,
		Value:       1,
		Stackable:   true,
		Quantity:    1,
	},
	"chicken_feather": {
		ID:          "chicken_feather",
		Name:        "Chicken Feather",
		Description: "A soft, white chicken feather.",
		Type:        ItemTypeMisc,
		Value:       1,
		Stackable:   true,
		Quantity:    1,
	},
	"raw_chicken": {
		ID:          "raw_chicken",
		Name:        "Raw Chicken",
		Description: "A freshly plucked chicken, ready to be cooked.",
		Type:        ItemTypeConsumable,
		Value:       5,
		Stackable:   true,
		Quantity:    1,
	},
	"leather_helmet": {
		ID:          "leather_helmet",
		Name:        "Leather Helmet",
		Description: "A sturdy helmet made of tanned leather.",
		Type:        ItemTypeArmor,
		Value:       50,
		Stackable:   false,
		Quantity:    1,
	},
	"leather_chest": {
		ID:          "leather_chest",
		Name:        "Leather Chestpiece",
		Description: "A protective chestpiece crafted from hardened leather.",
		Type:        ItemTypeArmor,
		Value:       75,
		Stackable:   false,
		Quantity:    1,
	},
	"leather_legs": {
		ID:          "leather_legs",
		Name:        "Leather Leggings",
		Description: "Flexible leather leggings that provide good protection.",
		Type:        ItemTypeArmor,
		Value:       60,
		Stackable:   false,
		Quantity:    1,
	},
	"leather_boots": {
		ID:          "leather_boots",
		Name:        "Leather Boots",
		Description: "Comfortable boots made from supple leather.",
		Type:        ItemTypeArmor,
		Value:       40,
		Stackable:   false,
		Quantity:    1,
	},
	"bone": {
		ID:          "bone",
		Name:        "Bone",
		Description: "A weathered bone from some unfortunate creature.",
		Type:        ItemTypeMisc,
		Value:       3,
		Stackable:   true,
		Quantity:    1,
	},
}

// CreateItem creates a new item from a template with the specified quantity
func CreateItem(itemID string, quantity int) *Item {
	template, exists := ItemTemplates[itemID]
	if !exists {
		return nil
	}

	item := template.Clone()
	item.Quantity = quantity
	return item
}
