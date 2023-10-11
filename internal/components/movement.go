package components

import "sync"

type MovementStatus int

const (
	Standing MovementStatus = iota
	Down
	Walking
	Running
)

type Movement struct {
	sync.RWMutex

	Status    MovementStatus
	Direction string
}
