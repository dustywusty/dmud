package components

// ConsumableEffect describes what eating/drinking a consumable item does. Like
// MountRegistry it is keyed by item ID in code, while the items themselves live
// in ItemTemplates.
type ConsumableEffect struct {
	ItemID      string
	RestoreHP   int
	RestoreEP   int    // endurance restored
	SelfMessage string // shown to the consumer
	AreaMessage string // shown to others; one %s = the consumer's name
}

// ConsumableRegistry holds every edible/drinkable item, keyed by item ID.
var ConsumableRegistry = map[string]*ConsumableEffect{
	"raw_chicken": {
		ItemID: "raw_chicken", RestoreEP: 20,
		SelfMessage: "You gnaw on the raw chicken. Gristly, but it puts some pep back in your step.",
		AreaMessage: "%s gnaws on a raw chicken.",
	},
	"bread": {
		ItemID: "bread", RestoreEP: 40,
		SelfMessage: "You eat the crusty bread and feel your second wind return.",
		AreaMessage: "%s tears into a loaf of bread.",
	},
	"stamina_draught": {
		ItemID: "stamina_draught", RestoreEP: 75,
		SelfMessage: "You down the stamina draught — energy surges back into your limbs.",
		AreaMessage: "%s drinks a fizzing green draught.",
	},
	"healing_potion": {
		ItemID: "healing_potion", RestoreHP: 50, RestoreEP: 15,
		SelfMessage: "You quaff the healing potion. Warmth knits your wounds and steadies your breath.",
		AreaMessage: "%s quaffs a swirling red potion.",
	},
}

// ConsumableFor returns the effect for an item ID, if it is consumable.
func ConsumableFor(itemID string) (*ConsumableEffect, bool) {
	c, ok := ConsumableRegistry[itemID]
	return c, ok
}
