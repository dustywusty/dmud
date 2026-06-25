package components

// Ware is a single item a vendor sells, priced in gold coins. MinRep optionally
// raises the faction requirement for this specific item above the vendor's
// baseline (0 means "use the vendor's MinRepToBuy").
type Ware struct {
	ItemID string
	Price  int
	MinRep int
}

// VendorDef describes a merchant NPC: what they sell, the faction standing
// required to buy, and how giving them a desired item raises that standing.
// Vendors are keyed by NPC template ID, mirroring how quests are keyed by NPCID.
type VendorDef struct {
	NPCTemplateID string
	ShopName      string

	// Faction gating.
	FactionID   string // faction this vendor's standing is tracked under
	MinRepToBuy int    // reputation required before the vendor will sell

	// Gift loop: giving DesiredItemID raises FactionID reputation.
	DesiredItemID string
	RepPerItem    int

	// Flavor lines.
	GreetingLine   string // shown on hail
	GiftAcceptLine string // shown when given the desired item
	GiftRejectLine string // shown when given anything else

	Wares []Ware
}

// FindWare returns the ware matching an item ID, if the vendor sells it.
func (v *VendorDef) FindWare(itemID string) (Ware, bool) {
	for _, w := range v.Wares {
		if w.ItemID == itemID {
			return w, true
		}
	}
	return Ware{}, false
}

// VendorForNPC returns the vendor definition for an NPC template, if any.
func VendorForNPC(templateID string) (*VendorDef, bool) {
	v, ok := VendorRegistry[templateID]
	return v, ok
}

// VendorRegistry holds all vendor definitions, keyed by NPC template ID.
var VendorRegistry = map[string]*VendorDef{
	"wylie": {
		NPCTemplateID:  "wylie",
		ShopName:       "Wylie's Riverside Paddock",
		FactionID:      "cinderhollow",
		MinRepToBuy:    40, // "Friend"
		DesiredItemID:  "cookie",
		RepPerItem:     5,
		GreetingLine:   "Oh! Hello hello, welcome to the paddock. Mind the lace, it's freshly dyed -- and mind the courser, she bites. Did you... happen to bring cookies from the bakehouse?",
		GiftAcceptLine: "Cookies?! For me?? Oh, you absolute treasure -- come, come, let me show you the horses.",
		GiftRejectLine: "That's very sweet of you, love, but Wylie only trades in cookies. Bring me cookies, yeah?",
		Wares: []Ware{
			{ItemID: "pony", Price: 80},                 // Friend (vendor baseline)
			{ItemID: "courser", Price: 200, MinRep: 80}, // Honored
		},
	},
}
