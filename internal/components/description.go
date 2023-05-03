package components

type Description interface {
	GetDescription() string
}

type DescriptionComponent struct {
	Description string
}

func (d *DescriptionComponent) GetDescription() string {
	return d.Description
}
