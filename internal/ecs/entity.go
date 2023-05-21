package ecs

import (
	"log"

	"github.com/google/uuid"
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

func NewEntityWithID(id string) Entity {
	log.Printf("Created entity %v", id)
	return Entity{
		ID: EntityID(id),
	}
}

func (e Entity) String() string {
	return string(e.ID)
}
