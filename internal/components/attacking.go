package components

type AttackingComponent struct {
	TargetID  string // The ID of the entity being attacked.
	MinDamage int    // The minimum amount of damage this entity can deal.
	MaxDamage int    // The maximum amount of damage this entity can deal.
}
