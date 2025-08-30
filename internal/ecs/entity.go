package ecs

import (
	"dmud/internal/common"

	"github.com/golang-module/carbon/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Entity struct {
	Components map[string]bool
	CreatedAt  carbon.Carbon
	ID         common.EntityID
	UpdatedAt  carbon.Carbon
}

func (e Entity) GetID() common.EntityID {
	return e.ID
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
		CreatedAt:  carbon.Now(),
		UpdatedAt:  carbon.Now(),
	}
}

func (e Entity) String() string {
	return string(e.ID)
}
