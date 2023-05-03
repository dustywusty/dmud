package components

type Name interface {
	GetName() string
}

type NameComponent struct {
	Name string
}

func (n *NameComponent) GetName() string {
	return n.Name
}
