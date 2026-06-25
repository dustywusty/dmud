package components

// Creation tracks a new ("ghost") character's progress through character
// creation. It's attached to un-manifested players and removed once they take
// shape. Transient — only the Player.Created flag is persisted.
type Creation struct {
	NamePicked  bool
	RacePicked  bool
	ClassPicked bool
}

func (c *Creation) Type() string { return "Creation" }

func NewCreation() *Creation { return &Creation{} }

// Done reports whether all three creation choices have been made.
func (c *Creation) Done() bool {
	return c.NamePicked && c.RacePicked && c.ClassPicked
}
