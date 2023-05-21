package ecs

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"dmud/internal/util"
)

type World struct {
	entities    map[EntityID]Entity
	components  map[EntityID]map[string]Component
	systems     []System
	entityMutex sync.Mutex
}

func NewWorld() *World {
	return &World{
		entities:   make(map[EntityID]Entity),
		components: make(map[EntityID]map[string]Component),
	}
}

func (w *World) AddEntity(entity Entity) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()
	w.entities[entity.ID] = entity
	w.components[entity.ID] = make(map[string]Component)
}

/* FindEntityByComponentPredicate finds an entity with a component matching the given predicate.
 * The predicate is a function that takes an interface{} and returns a bool. The interface{} is the component
 * that is being checked. The bool is whether or not the component matches the predicate.
 */

func (w *World) FindEntityByComponentPredicate(componentType string, predicate func(interface{}) bool) (Entity, error) {
	for entityID, components := range w.components {
		component, ok := components[componentType]
		if ok && predicate(component) {
			if entity, exists := w.entities[entityID]; exists {
				return entity, nil
			}
			return Entity{}, fmt.Errorf("no entity found matching the entity id")
		}
	}
	return Entity{}, fmt.Errorf("no entity found matching the predicate")
}

func (w *World) FindEntitiesByComponentPredicate(componentType string, predicate func(interface{}) bool) ([]Entity, error) {
	entities := make([]Entity, 0)
	for entityID, components := range w.components {
		component, ok := components[componentType]
		if ok && predicate(component) {
			if entity, exists := w.entities[entityID]; exists {
				entities = append(entities, entity)
			} else {
				return nil, fmt.Errorf("no entity found matching the entity ID")
			}
		}
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities found matching the predicate")
	}
	return entities, nil
}

func (w *World) FindEntity(id EntityID) (Entity, error) {
	entity, ok := w.entities[id]
	if !ok {
		return Entity{}, errors.New("no entity found with that ID")
	}
	return entity, nil
}

func (w *World) RemoveEntity(entityID EntityID) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()
	delete(w.entities, entityID)
	delete(w.components, entityID)
}

func (w *World) AddComponent(entity Entity, component Component) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()
	componentName := reflect.TypeOf(component).Elem().Name()
	w.components[entity.ID][componentName] = component
}

func (w *World) AddSystem(system System) {
	w.systems = append(w.systems, system)
}

func (w *World) GetComponent(entityID EntityID, componentName string) (Component, error) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()
	if components, ok := w.components[entityID]; ok {
		if component, ok := components[componentName]; ok {
			return component, nil
		}
		return nil, errors.New("component not found")
	}
	return nil, errors.New("entity not found")
}

func (w *World) Update() {
	deltaTime := util.CalculateDeltaTime()
	for _, system := range w.systems {
		system.Update(w, deltaTime)
	}
}
