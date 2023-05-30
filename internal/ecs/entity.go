package ecs

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type EntityID string

type Entity struct {
	ID EntityID
}

func NewEntity() Entity {
	_uuid := uuid.New()
	log.Info().Msgf("Created entity %v", _uuid.String())
	return Entity{
		ID: EntityID(_uuid.String()),
	}
}

func NewEntityWithID(id string) Entity {
	log.Info().Msgf("Created entity %v", id)
	return Entity{
		ID: EntityID(id),
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

func (e Entity) String() string {
	return string(e.ID)
}
