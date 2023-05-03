package ecs

import (
	"sync"
)

type World struct {
	entities    map[EntityID]*Entity
	components  map[string]map[EntityID]interface{}
	systems     []System
	entitiesMux sync.RWMutex
}

func NewWorld() *World {
	return &World{
		entities:   make(map[EntityID]*Entity),
		components: make(map[string]map[EntityID]interface{}),
	}
}

func (w *World) AddSystem(system System) {
	system.SetWorld(w)
	w.systems = append(w.systems, system)
}

func (w *World) Update(deltaTime float64) {
	for _, system := range w.systems {
		system.Update(deltaTime, w)
	}
}

func (w *World) CreateEntity() *Entity {
	entity := NewEntity()
	w.entitiesMux.Lock()
	defer w.entitiesMux.Unlock()
	w.entities[entity.ID] = entity
	return entity
}

func (w *World) GetEntity(id EntityID) *Entity {
	w.entitiesMux.RLock()
	defer w.entitiesMux.RUnlock()
	return w.entities[id]
}

func (w *World) AddComponent(entity *Entity, component interface{}) {
	componentName := GetComponentName(component)
	if _, ok := w.components[componentName]; !ok {
		w.components[componentName] = make(map[EntityID]interface{})
	}
	w.components[componentName][entity.ID] = component
}

func (w *World) GetComponent(entity *Entity, componentName string) interface{} {
	components, ok := w.components[componentName]
	if !ok {
		return nil
	}
	return components[entity.ID]
}

func (w *World) GetEntitiesWithComponents(componentNames ...string) []*Entity {
	var result []*Entity
	for id, entity := range w.entities {
		includesAll := true
		for _, componentName := range componentNames {
			if _, ok := w.components[componentName][id]; !ok {
				includesAll = false
				break
			}
		}
		if includesAll {
			result = append(result, entity)
		}
	}
	return result
}
