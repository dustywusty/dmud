// internal/components/interfaces.go
package components

import "dmud/internal/common"

type WorldLike interface {
	FindEntitiesByComponentPredicate(componentType string, predicate func(interface{}) bool) ([]EntityLike, error)
	GetComponent(entityID common.EntityID, componentType string) (interface{}, error)
	RemoveComponent(entityID common.EntityID, componentType string) error
	RemoveEntity(entityID common.EntityID) error
}

type EntityLike interface {
	GetID() common.EntityID
}
