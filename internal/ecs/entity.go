package ecs

import (
	"github.com/google/uuid"
	"log"
)

type EntityID string

type Entity struct {
	ID EntityID
}

func NewEntity() Entity {
	_uuid := uuid.New()
	log.Printf("Created entity %v", _uuid.String())
	return Entity{
		ID: EntityID(_uuid.String()),
	}
}

func (e Entity) String() string {
	return string(e.ID)
}
