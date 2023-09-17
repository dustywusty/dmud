package ecs

import (
	"dmud/internal/common"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Entity struct {
	ID         common.EntityID
	Components map[string]bool
}

func NewEntity(ids ...string) Entity {
	var id string

	if len(ids) > 0 {
		id = ids[0]
	} else {
		_uuid := uuid.New()
		id = _uuid.String()
	}

	log.Info().Msgf("Created entity %v", id)

	return Entity{
		ID:         common.EntityID(id),
		Components: make(map[string]bool),
	}
}

func (e Entity) String() string {
	return string(e.ID)
}
