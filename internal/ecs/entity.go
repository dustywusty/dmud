package ecs

import (
	"sync/atomic"
)

type EntityID uint64

type Entity struct {
	ID EntityID
}

var entityIDCounter EntityID

func NewEntity() Entity {
	return Entity{
		ID: EntityID(atomic.AddUint64((*uint64)(&entityIDCounter), 1)),
	}
}
