package ecs

import (
	"fmt"
	"reflect"
	"sync"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/util"

	"github.com/rs/zerolog/log"
)

type World struct {
	components     map[common.EntityID]map[string]Component
	componentMutex sync.RWMutex

	entities    map[common.EntityID]Entity
	entityMutex sync.RWMutex

	elapsedTime float64

	systems []System
}

// -----------------------------------------------------------------------------

func (w *World) AddComponent(entity *Entity, component Component) {
	w.componentMutex.Lock()
	defer w.componentMutex.Unlock()

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

	log.Info().Msgf("Added component %s to entity %s", componentName, entity.ID)
}

// -----------------------------------------------------------------------------

func (w *World) AddEntity(entity Entity) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()

	w.componentMutex.Lock()
	defer w.componentMutex.Unlock()

	w.entities[entity.ID] = entity
	w.components[entity.ID] = make(map[string]Component)

	log.Info().Msgf("Added entity %s", entity.ID)
}

// -----------------------------------------------------------------------------

func (w *World) AddSystem(system System) {
	w.systems = append(w.systems, system)
}

// -----------------------------------------------------------------------------

func (w *World) Components() map[common.EntityID]map[string]Component {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()

	return w.components
}

// -----------------------------------------------------------------------------

func (w *World) Entities() map[common.EntityID]Entity {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()

	return w.entities
}

// -----------------------------------------------------------------------------

func (w *World) FindEntity(id common.EntityID) (Entity, error) {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()

	entity, ok := w.entities[id]

	if !ok {
		return Entity{}, nil
	}

	return entity, nil
}

// -----------------------------------------------------------------------------

func (w *World) FindEntitiesByComponentPredicate(componentType string, predicate func(interface{}) bool) ([]Entity, error) {
	w.componentMutex.RLock()
	defer w.componentMutex.RUnlock()

	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()

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

// -----------------------------------------------------------------------------

func (w *World) GetComponent(entityID common.EntityID, componentName string) (interface{}, error) {
	w.componentMutex.RLock()
	defer w.componentMutex.RUnlock()

	if components, ok := w.components[entityID]; ok {
		if component, ok := components[componentName]; ok {
			return component, nil
		}
		return nil, fmt.Errorf("component %s not found for entity %s", componentName, entityID)
	}

	return nil, fmt.Errorf("entity %s not found", entityID)
}

// -----------------------------------------------------------------------------

func (w *World) RemoveComponent(entityID common.EntityID, componentName string) {
	w.componentMutex.Lock()
	defer w.componentMutex.Unlock()

	if _, ok := w.components[entityID]; ok {
		delete(w.components[entityID], componentName)
		if len(w.components[entityID]) == 0 {
			delete(w.components, entityID)
		}
	} else {
		log.Error().Msgf("Entity %s does not have component %s", entityID, componentName)
	}

	log.Info().Msgf("Removed component %s from entity %s", componentName, entityID)
}

// -----------------------------------------------------------------------------

func (w *World) RemoveEntity(entityID common.EntityID) {
	if playerComponent, err := w.GetComponent(entityID, "Player"); err == nil {
		player, ok := playerComponent.(*components.Player)
		if !ok {
			log.Error().Msgf("Error type asserting Player for player %s", entityID)
		} else {
			if player.Room != nil {
				player.Room.RemovePlayer(player)
			}
		}
	}

	w.entityMutex.Lock()
	delete(w.entities, entityID)
	w.entityMutex.Unlock()

	w.componentMutex.Lock()
	delete(w.components, entityID)
	w.componentMutex.Unlock()

	log.Info().Msgf("Removed entity %s", entityID)
}

// -----------------------------------------------------------------------------

func (w *World) Update() {
	deltaTime := util.CalculateDeltaTime()

	w.elapsedTime += deltaTime

	if w.elapsedTime >= 0.1 {
		for _, system := range w.systems {
			system.Update(w, deltaTime)
		}

		w.elapsedTime = 0
	}
}

// -----------------------------------------------------------------------------

func NewWorld() *World {
	world := &World{
		entities:   make(map[common.EntityID]Entity),
		components: make(map[common.EntityID]map[string]Component),
	}

	rooms := loadRoomsFromFile("./resources/rooms.json")

	for _, room := range rooms {
		roomEntity := NewEntity(room.ID)
		roomComponent := &components.Room{
			Description: room.Description,
		}
		world.AddEntity(roomEntity)
		world.AddComponent(&roomEntity, roomComponent)
	}

	for _, room := range rooms {
		component, err := world.GetComponent(common.EntityID(room.ID), "Room")
		if err != nil {
			log.Error().Err(err).Msgf("Could not get Room for room %s", room.ID)
			continue
		}

		roomComponent, ok := component.(*components.Room)
		if !ok {
			log.Error().Msgf("Error type asserting Room for room %s", room.ID)
			continue
		}

		for direction, roomID := range room.Exits {
			exitRoomUntyped, err := world.GetComponent(common.EntityID(roomID), "Room")
			if err != nil {
				log.Error().Err(err).Msgf("Could not get Room for exit room %s", roomID)
				continue
			}

			exitRoom, ok := exitRoomUntyped.(*components.Room)
			if !ok {
				log.Error().Msgf("Error type asserting Room for exit room %s", roomID)
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

// -----------------------------------------------------------------------------

type Room struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

// -----------------------------------------------------------------------------

func loadRoomsFromFile(filename string) []Room {
	var rooms []Room
	if err := util.ParseJSON(filename, &rooms); err != nil {
		log.Error().Err(err).Msg("")
	}
	return rooms
}
