package components

import (
	"dmud/internal/common"
)

type PlayerComponent struct {
	Client common.Client
	Name   string
	Room   *RoomComponent
}
