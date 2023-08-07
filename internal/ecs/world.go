package ecs

import (
	"errors"
	"reflect"
	"sync"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/util"

	"github.com/rs/zerolog/log"
)

type World struct {
	components          map[common.EntityID]map[string]Component
	componentToEntities map[string]map[common.EntityID]Entity

	entities    map[common.EntityID]Entity
	entityMutex sync.RWMutex

	systems []System
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func (w *World) AddComponent(entity *Entity, component Component) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()

	componentName := reflect.TypeOf(component).Elem().Name()

	if _, ok := w.components[entity.ID]; ok {
		w.components[entity.ID][componentName] = component
	} else {
		w.components[entity.ID] = make(map[string]Component)
		w.components[entity.ID][componentName] = component
	}

	entity.Components[componentName] = true
}

func (w *World) AddEntity(entity Entity) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()
	w.entities[entity.ID] = entity
	w.components[entity.ID] = make(map[string]Component)
}

func (w *World) AddSystem(system System) {
	w.systems = append(w.systems, system)
}

func (w *World) Components() map[common.EntityID]map[string]Component {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()
	return w.components
}

func (w *World) Entities() map[common.EntityID]Entity {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()
	return w.entities
}

func (w *World) FindEntity(id common.EntityID) (Entity, error) {
	entity, ok := w.entities[id]
	if !ok {
		return Entity{}, nil
	}
	return entity, nil
}

func (w *World) FindEntitiesByComponentPredicate(componentType string, predicate func(interface{}) bool) ([]Entity, error) {
	entities := make([]Entity, 0)
	for entityID, components := range w.components {
		component, ok := components[componentType]
		if ok && predicate(component) {
			if entity, exists := w.entities[entityID]; exists {
				entities = append(entities, entity)
			}
		}
	}
	if len(entities) == 0 {
		return nil, nil
	}
	return entities, nil
}

func (w *World) GetComponent(entityID common.EntityID, componentName string) (Component, error) {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()
	if components, ok := w.components[entityID]; ok {
		if component, ok := components[componentName]; ok {
			return component, nil
		}
		return nil, errors.New("component not found")
	}
	return nil, errors.New("entity not found")
}

func (w *World) RemoveEntity(entityID common.EntityID) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()
	delete(w.entities, entityID)
	delete(w.components, entityID)
}

func (w *World) Update() {
	deltaTime := util.CalculateDeltaTime()
	for _, system := range w.systems {
		system.Update(w, deltaTime)
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func NewWorld() *World {
	world := &World{
		entities:   make(map[common.EntityID]Entity),
		components: make(map[common.EntityID]map[string]Component),
	}

	rooms := loadRoomsFromFile("./resources/rooms.json")

	for _, room := range rooms {
		roomEntity := NewEntity(room.ID)
		roomComponent := &components.RoomComponent{
			Description: room.Description,
		}
		world.AddEntity(roomEntity)
		world.AddComponent(&roomEntity, roomComponent)
	}

	for _, room := range rooms {
		component, err := world.GetComponent(common.EntityID(room.ID), "RoomComponent")
		if err != nil {
			log.Error().Err(err).Msgf("Could not get RoomComponent for room %s", room.ID)
			continue
		}

		roomComponent, ok := component.(*components.RoomComponent)
		if !ok {
			log.Error().Msgf("Error type asserting RoomComponent for room %s", room.ID)
			continue
		}

		for direction, roomID := range room.Exits {
			exitRoomComponent, err := world.GetComponent(common.EntityID(roomID), "RoomComponent")
			if err != nil {
				log.Error().Err(err).Msgf("Could not get RoomComponent for exit room %s", roomID)
				continue
			}

			exitRoom, ok := exitRoomComponent.(*components.RoomComponent)
			if !ok {
				log.Error().Msgf("Error type asserting RoomComponent for exit room %s", roomID)
				continue
			}

			roomComponent.Exits = append(roomComponent.Exits, components.Exit{
				Direction: direction,
				RoomID:    roomID,
				Room:      exitRoom,
			})
		}
	}

	return world
}

// ..

type Room struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

func loadRoomsFromFile(filename string) []Room {
	var rooms []Room
	if err := util.ParseJSON(filename, &rooms); err != nil {
		log.Error().Err(err).Msg("")
	}
	return rooms
}
