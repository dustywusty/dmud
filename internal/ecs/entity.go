package ecs

import (
	"sync/atomic"
)

type EntityID uint64

var entityIDCounter EntityID

func getNextEntityID() EntityID {
	return EntityID(atomic.AddUint64((*uint64)(&entityIDCounter), 1))
}

type Entity struct {
	ID EntityID
}

func NewEntity() *Entity {
	return &Entity{
		ID: getNextEntityID(),
	}
}
